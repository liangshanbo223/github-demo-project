package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/liangshanbo223/github-demo-project/config"
	"github.com/liangshanbo223/github-demo-project/logger"
)

type PanelService struct {
}

func (s *PanelService) RestartPanel(delay time.Duration) error {
	p, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		return err
	}
	go func() {
		time.Sleep(delay)
		if runtime.GOOS == "windows" {
			err = p.Kill()
		} else {
			err = p.Signal(syscall.SIGHUP)
		}
		if err != nil {
			logger.Error("send signal SIGHUP failed:", err)
		}
	}()
	return nil
}

func (s *PanelService) getBackupDir() string {
	dir := "/usr/local/s-ui/backup/bin"
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fallback to db folder path backup
		dir = filepath.Join(config.GetDBFolderPath(), "backup", "bin")
		os.MkdirAll(dir, 0755)
	}
	return dir
}

func (s *PanelService) HotSwapAndRestart() error {
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = os.Args[0]
	}

	logger.Info("Hot-swapping and restarting process: ", binaryPath)
	time.Sleep(1 * time.Second)

	err = syscall.Exec(binaryPath, os.Args, os.Environ())
	return err
}

func (s *PanelService) UpdateSystem(url string) error {
	tempTar := filepath.Join(os.TempDir(), "s-ui-update.tar.gz")
	tempBin := filepath.Join(os.TempDir(), "sui_new")

	logger.Info("Downloading update from: ", url)
	err := downloadFile(url, tempTar)
	if err != nil {
		return fmt.Errorf("下载更新失败: %w", err)
	}
	defer os.Remove(tempTar)

	logger.Info("Extracting update...")
	err = extractTarGz(tempTar, tempBin)
	if err != nil {
		// Check if the URL is just a raw binary instead of tar.gz
		logger.Info("Tar extraction failed, trying as raw binary...")
		err = downloadFile(url, tempBin)
		if err != nil {
			return fmt.Errorf("解析/提取更新失败: %w", err)
		}
		os.Chmod(tempBin, 0755)
	}
	defer os.Remove(tempBin)

	// Get current running executable path
	currentPath, err := os.Executable()
	if err != nil {
		currentPath = os.Args[0]
	}

	// Backup current binary
	backupDir := s.getBackupDir()
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("sui_backup_%s", timestamp))
	
	logger.Info("Backing up current binary to: ", backupPath)
	err = copyFile(currentPath, backupPath)
	if err != nil {
		return fmt.Errorf("备份当前二进制失败: %w", err)
	}

	// Hot-replace running binary
	logger.Info("Replacing running binary...")
	oldPath := currentPath + ".old"
	os.Remove(oldPath) // remove any leftover
	
	err = os.Rename(currentPath, oldPath)
	if err != nil {
		return fmt.Errorf("重命名当前二进制失败: %w", err)
	}

	err = copyFile(tempBin, currentPath)
	if err != nil {
		// Rollback rename if copy fails
		os.Rename(oldPath, currentPath)
		return fmt.Errorf("写入新二进制失败: %w", err)
	}
	os.Chmod(currentPath, 0755)
	os.Remove(oldPath)

	logger.Info("Update applied successfully. Restarting process...")
	go func() {
		err := s.HotSwapAndRestart()
		if err != nil {
			logger.Error("Hot restart failed: ", err.Error())
		}
	}()

	return nil
}

func (s *PanelService) RollbackSystem() error {
	backupDir := s.getBackupDir()
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}

	var backups []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "sui_backup_") {
			backups = append(backups, filepath.Join(backupDir, f.Name()))
		}
	}

	if len(backups) == 0 {
		return fmt.Errorf("未找到任何备份版本")
	}

	// Sort backups (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	newestBackup := backups[0]

	logger.Info("Rolling back to backup: ", newestBackup)

	currentPath, err := os.Executable()
	if err != nil {
		currentPath = os.Args[0]
	}

	oldPath := currentPath + ".old"
	os.Remove(oldPath)

	err = os.Rename(currentPath, oldPath)
	if err != nil {
		return fmt.Errorf("重命名当前二进制失败: %w", err)
	}

	err = copyFile(newestBackup, currentPath)
	if err != nil {
		os.Rename(oldPath, currentPath)
		return fmt.Errorf("回滚还原二进制失败: %w", err)
	}
	os.Chmod(currentPath, 0755)
	os.Remove(oldPath)

	logger.Info("Rollback applied successfully. Restarting process...")
	go func() {
		err := s.HotSwapAndRestart()
		if err != nil {
			logger.Error("Hot restart failed: ", err.Error())
		}
	}()

	return nil
}

func (s *PanelService) ListBackups() ([]string, error) {
	backupDir := s.getBackupDir()
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var backups []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "sui_backup_") {
			backups = append(backups, f.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	return backups, nil
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(src string, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg && (header.Name == "sui" || strings.HasSuffix(header.Name, "/sui")) {
			outFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer outFile.Close()
			_, err = io.Copy(outFile, tr)
			return err
		}
	}
	return fmt.Errorf("sui binary not found in archive")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
