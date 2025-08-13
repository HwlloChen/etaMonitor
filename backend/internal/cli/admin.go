package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"etamonitor/internal/auth"
	"etamonitor/internal/models"

	"golang.org/x/term"
	"gorm.io/gorm"
)

// SetupAdmin 交互式设置管理员账户
func SetupAdmin(db *gorm.DB, isFirstTime bool) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n=== 设置管理员账户 ===")
	if isFirstTime {
		fmt.Println("首次启动，请设置管理员账户")
	}

	// 读取用户名
	fmt.Print("请输入管理员用户名: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取用户名失败: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}

	// 读取密码
	fmt.Print("请输入管理员密码: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("读取密码失败: %w", err)
	}
	fmt.Println()

	if len(password) < 6 {
		return fmt.Errorf("密码长度不能小于6位")
	}

	// 确认密码
	fmt.Print("请再次输入密码: ")
	confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("读取确认密码失败: %w", err)
	}
	fmt.Println()

	if string(password) != string(confirmPassword) {
		return fmt.Errorf("两次输入的密码不一致")
	}

	// 生成加盐哈希密码
	hashedPassword, err := auth.GeneratePasswordHash(string(password))
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	// 更新或创建管理员账户
	var admin models.User
	result := db.Where("role = ?", "admin").First(&admin)

	if result.Error == nil {
		// 更新现有管理员
		admin.Username = username
		admin.Password = hashedPassword
		if err := db.Save(&admin).Error; err != nil {
			return fmt.Errorf("更新管理员账户失败: %w", err)
		}
		fmt.Println("\n管理员账户已更新")
	} else if result.Error == gorm.ErrRecordNotFound {
		// 创建新管理员
		admin = models.User{
			Username: username,
			Password: hashedPassword,
			Role:     "admin",
		}
		if err := db.Create(&admin).Error; err != nil {
			return fmt.Errorf("创建管理员账户失败: %w", err)
		}
		fmt.Println("\n管理员账户已创建")
	} else {
		return fmt.Errorf("查询管理员账户失败: %w", result.Error)
	}

	fmt.Printf("\n=== 管理员账户信息 ===\n")
	fmt.Printf("用户名: %s\n", username)
	fmt.Println("密码已设置")
	fmt.Println("=====================")

	return nil
}
