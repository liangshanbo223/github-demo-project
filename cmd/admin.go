package cmd

import (
	"fmt"

	"github.com/liangshanbo223/github-demo-project/config"
	"github.com/liangshanbo223/github-demo-project/database"
	"github.com/liangshanbo223/github-demo-project/service"
)

func resetAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	userService := service.UserService{}
	err = userService.UpdateFirstUser("admin", "admin")
	if err != nil {
		fmt.Println("重置管理员凭据失败：", err)
	} else {
		fmt.Println("重置管理员凭据成功")
	}
}

func updateAdmin(username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("重置管理员凭据失败：", err)
		} else {
			fmt.Println("重置管理员凭据成功")
		}
	}
}

func showAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}
	userService := service.UserService{}
	userModel, err := userService.GetFirstUser()
	if err != nil {
		fmt.Println("获取当前用户信息失败，错误信息：", err)
	}
	username := userModel.Username
	userpasswd := userModel.Password
	if (username == "") || (userpasswd == "") {
		fmt.Println("当前用户名或密码为空")
	}
	fmt.Println("首个管理员凭据：")
	fmt.Println("\t用户名:\t", username)
	fmt.Println("\t密码:\t", userpasswd)
}
