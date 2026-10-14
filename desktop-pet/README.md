# Hello Kitty 桌面小伙伴 · V3 原图还原版

参考用户提供的 Hello Kitty 图片制作，保留头上的粉色波点蝴蝶结，去掉身体下方的大蝴蝶结。V3 使用完整图等比绘制，保留参考图的五官、头身比例、斜坐姿和单个前脚爱心。透明背景，适用于 Windows 10 / 11 64 位。

[下载 V3 免安装版 ZIP](../deliverables/hello-kitty-desktop-pet-v3-windows.zip?raw=true) · [直接下载 V3 EXE](../deliverables/HelloKittyPet-V3.exe?raw=true) · [观看互动预览](../deliverables/hello-kitty-pet-v3-preview.gif?raw=true) · [查看外观](../deliverables/hello-kitty-pet-v3-look.png?raw=true)

保留 V2.1 的全屏自动避让：应用覆盖整个屏幕且任务栏也被遮住时，宠物自动隐藏；退出全屏后自动回到原来的位置。桌面、普通窗口、浏览器最大化，以及任务栏仍可见的铺满窗口，都保持显示。

完整造型配合柔和眨眼、小幅挥手、轻轻歪头和整体起伏，保留熟悉的坐姿。鼠标靠近时出现快捷按钮，可以摸摸、喂食、玩球或让它安静睡觉。

## 使用

**升级时先右键退出旧版，再启动新版。** 解压 ZIP，双击 `HelloKittyPet-V3.exe`。首次启动时宠物出现在屏幕右下角，默认始终置顶。程序运行不需要联网或管理员权限，也不会自动添加开机启动。

自动隐藏与恢复通常在半秒内完成，不会抢走正在使用的窗口焦点。多显示器时只检查宠物所在屏幕；通过菜单手动收起的宠物不会因退出全屏而自行出现。Windows 开启任务栏自动隐藏时，普通带边框的最大化窗口仍保持显示。若曾关闭“始终显示在最前面”，需重新开启才能让宠物出现在浏览器等窗口上方。

| 操作 | 效果 |
| --- | --- |
| 单击宠物 | 摸摸、爱心与中文回应 |
| 按住左键拖动 | 移动到喜欢的位置 |
| 鼠标靠近 | 显示爱心 / 饼干 / 小球 / 月亮按钮 |
| 点击爱心 / 饼干 / 小球 / 月亮 | 摸摸 / 喂食 / 玩球 / 睡眠与唤醒 |
| 右键 → 挥挥小手 / 歪歪头 | 轻轻抬手，或整体小幅侧倾 |
| 右键 → 跟着鼠标走 | 开关左右跟随移动，靠近目标后停下 |
| 右键 → 自己散散步 | 开关闲逛，鼠标靠近时停下方便互动 |
| 右键 → 喂一口零食 | 小饼干、轻轻点头与碎屑 |
| 右键 → 一起玩耍 | 小幅摇摆、弹跳小球与星星 |
| 右键 → 让它睡一会儿 | 闭眼轻轻起伏、飘动的 Z；点击可以唤醒 |
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
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H windowsgui" -o HelloKittyPet-V3.exe .
```

Windows PowerShell：

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w -H windowsgui" -o HelloKittyPet-V3.exe .
```

原生 Win32 窗口和绘图使用 Go 标准库，无需安装浏览器运行时。应用图标与清单的 `rsrc_windows_amd64.syso` 已包含在源码中；修改图标后可使用 `github.com/akavel/rsrc` 重新生成。字体图集由 `make_font.py` 构建，所用 Noto Sans CJK 的许可证见 `FONT-LICENSE.txt`。

已验证行为状态测试、透明像素与动画测试、Windows 64 位编译及 Wine 下的实际窗口交互。保留自 V2.1 的全屏策略已通过全屏覆盖、任务栏保持可见、退出全屏恢复、保留手动隐藏和恢复时不抢焦点的 Win32 窗口集成测试；未在实体 Windows 电脑或抖音客户端上测试。集成测试仅在 Wine 且设置 `PET_FULLSCREEN_INTEGRATION=1` 时运行，以免影响真实桌面。
