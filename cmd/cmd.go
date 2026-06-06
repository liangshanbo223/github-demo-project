package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/admin8800/s-ui/cmd/migration"
	"github.com/admin8800/s-ui/config"
)

func ParseCmd() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "显示版本信息")

	adminCmd := flag.NewFlagSet("admin", flag.ExitOnError)
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)

	var username string
	var password string
	var port int
	var path string
	var subPort int
	var subPath string
	var reset bool
	var show bool
	settingCmd.BoolVar(&reset, "reset", false, "重置所有设置")
	settingCmd.BoolVar(&show, "show", false, "显示当前设置")
	settingCmd.IntVar(&port, "port", 0, "设置面板端口")
	settingCmd.StringVar(&path, "path", "", "设置面板根路径")
	settingCmd.IntVar(&subPort, "subPort", 0, "设置订阅端口")
	settingCmd.StringVar(&subPath, "subPath", "", "设置订阅根路径")

	adminCmd.BoolVar(&show, "show", false, "显示首个管理员的凭据")
	adminCmd.BoolVar(&reset, "reset", false, "重置首个管理员的凭据")
	adminCmd.StringVar(&username, "username", "", "设置登录用户名")
	adminCmd.StringVar(&password, "password", "", "设置登录密码")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("命令列表:")
		fmt.Println("    admin          设置/重置/显示首个管理员的凭据")
		fmt.Println("    uri            显示面板访问 URI")
		fmt.Println("    migrate        从旧版本迁移数据库")
		fmt.Println("    setting        设置/重置/显示面板参数配置")
		fmt.Println("    node           运行子节点同步守护进程")
		fmt.Println()
		adminCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}

	flag.Parse()
	if showVersion {
		fmt.Println("S-UI 面板\t", config.GetVersion())
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, dep := range info.Deps {
				if dep.Path == "github.com/sagernet/sing-box" {
					fmt.Println("Sing-Box 内核\t", dep.Version)
					break
				}
			}
		}
		return
	}

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case "admin":
		err := adminCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showAdmin()
		case reset:
			resetAdmin()
		default:
			updateAdmin(username, password)
			showAdmin()
		}

	case "uri":
		getPanelURI()

	case "migrate":
		migration.MigrateDb()

	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showSetting()
		case reset:
			resetSetting()
		default:
			updateSetting(port, path, subPort, subPath)
			showSetting()
		}
	case "node":
		RunNode()
	default:
		fmt.Println("无效的子命令")
		flag.Usage()
	}
}
