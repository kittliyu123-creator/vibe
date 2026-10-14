# Hello Kitty 桌面小伙伴 · V2

参考用户提供的 Hello Kitty 图片制作，保留头上的粉色波点蝴蝶结，去掉身体下方的大蝴蝶结。透明背景，适用于 Windows 10 / 11 64 位。

[下载 V2 免安装版 ZIP](../deliverables/hello-kitty-desktop-pet-v2-windows.zip?raw=true) · [直接下载 V2 EXE](../deliverables/HelloKittyPet-V2.exe?raw=true) · [观看互动预览](../deliverables/hello-kitty-pet-v2-preview.gif?raw=true) · [观看行走预览](../deliverables/hello-kitty-pet-v2-walking.gif?raw=true)

V2 将头、躯干、双手与双脚分别绘制和运动，加入挥手、伸懒腰、交替迈步、平滑转头与眨眼。鼠标靠近时出现快捷按钮，眼睛会看向鼠标；喂食时会用双手捧起小饼干。

## 使用

**升级时先右键退出旧版，再启动新版。** 解压 ZIP，双击 `HelloKittyPet-V2.exe`。首次启动时宠物出现在屏幕右下角，默认始终置顶。程序运行不需要联网或管理员权限，也不会自动添加开机启动。

| 操作 | 效果 |
| --- | --- |
| 单击宠物 | 摸摸、爱心与中文回应 |
| 按住左键拖动 | 移动到喜欢的位置 |
| 鼠标靠近 | 眼神跟随，显示爱心 / 饼干 / 小球 / 月亮按钮 |
| 点击爱心 / 饼干 / 小球 / 月亮 | 摸摸 / 喂食 / 玩球 / 睡眠与唤醒 |
| 右键 → 挥手 / 伸懒腰 | 独立的手臂、头部动作 |
| 右键 → 跟着鼠标走 | 开关左右跟随，靠近目标后停下 |
| 右键 → 自己散散步 | 开关闲逛，鼠标靠近时停下方便互动 |
| 右键 → 喂一口零食 | 双手捧饼干、进食动作与碎屑 |
| 右键 → 一起玩耍 | 挥爪、踢脚、弹跳小球与星星 |
| 右键 → 让它睡一会儿 | 闭眼呼吸、飘动的 Z；点击可以唤醒 |
| 右键 → 小号 / 标准 / 大号 | 调整显示尺寸 |
| 右键 → 暂时收起到托盘 | 隐藏；点任务栏右下角的宠物图标恢复 |
| 右键 → 退出小伙伴 | 完全退出 |

托盘图标可能在 Windows 右下角的上箭头内。找不到宠物时，再双击 EXE 会召回现有宠物，不会重复创建。

位置、大小、置顶及移动偏好会保存，下次启动自动恢复；默认关闭跟随和闲逛。安静待机和睡觉时降低绘制频率，点击互动保持流畅。单击宠物不会抢走正在输入的窗口焦点。

设置保存在 `%APPDATA%\HelloKittyDesktopPet\settings.json`，退出程序后删除此文件即可恢复默认设置。

## 编译

安装 Go 1.25 或更新版本，在此目录运行：

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H windowsgui" -o HelloKittyPet-V2.exe .
```

Windows PowerShell：

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w -H windowsgui" -o HelloKittyPet-V2.exe .
```

原生 Win32 窗口和绘图使用 Go 标准库，无需安装浏览器运行时。应用图标与清单的 `rsrc_windows_amd64.syso` 已包含在源码中；修改图标后可使用 `github.com/akavel/rsrc` 重新生成。字体图集由 `make_font.py` 构建，所用 Noto Sans CJK 的许可证见 `FONT-LICENSE.txt`。

已验证行为状态测试、透明像素与动画测试、Windows 64 位编译及 Wine 下的实际窗口交互；未在实体 Windows 电脑上测试。
