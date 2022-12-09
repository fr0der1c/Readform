# Readform

该程序可以将付费新闻网站的**完整文章内容**发送到您的 [Readwise Reader](https://readwise.io/read) 的 feed 流中，以帮助您获得统一的阅读工作流。将来可能会支持 RSS 输出。

目前支持的网站：
- 端传媒
- 财新

未来将支持：
- 华尔街日报
- FT中文网

## 为什么创建 Readform？
市面上有许多基于订阅制的高质量媒体。我非常尊重他们的工作。不过，我相信订户有权以他/她喜欢的不同形式阅读，例如 RSS 阅读器。 **专业读者有自己定制化的阅读工作流。 对文章收费的“专业”媒体应该尊重读者自己的选择。** 由于这些网站没有官方的全文 RSS 支持，我决定自己制作一个。

目前，我使用 Readwise Reader 作为我的 RSS 阅读器（它仍处于测试阶段，您可向官方发送邮件询问是否可以加入测试），所以我进行了 Readwise Reader 输出集成。 将来也可能支持 RSS 输出。

该项目的最终目标是推动这些网站为其订户提供官方全文 RSS。在此之前，让我们使用这个程序吧！

## 这个程序是如何工作的？
该程序不断使用网站的 RSS 源获取最新文章。当有新文章时，它会模拟浏览器（使用 Chromedriver 和 Selenium）并使用您的凭据登录以获取完整的 HTML 内容。 懒加载的图片会被妥善处理，不用担心图片丢失。 该程序将使用官方 Reader API 将文章 URL 及其 HTML 内容发送到 Readwise Reader，因此您可以在您的 Reader 的 feed 部分中看到它们。


## 快速开始
Readform 不是基于云的服务，您需要在自己的机器上运行它。这将为您提供最高程度的安全性，因为使用此程序需要您的网站的用户名和密码。您可以在本地设备（PC、Mac、NAS、Raspberry Pi 等）上安装 Readform 或将其部署在 VPS 上。

推荐使用 Docker 运行 Readform。 如果您的计算机上没有 Docker，您可以[在此处下载](https://docs.docker.com/get-docker/)。

请注意，您需要确保程序所处的网络环境可以顺畅访问您希望订阅的网站。

1. 在终端中使用以下命令以在 Docker 中运行此程序。运行后，您将看到一个 docker container ID 的输出：
     ```
     docker run --restart=always -d \
         -e READFORM_WEBSITES=the_initium,caixin\
         -e THE_INITIUM_USERNAME=[你的用户名] \
         -e THE_INITIUM_PASSWORD=[你的密码] \
         -e CAIXIN_USERNAME=[你的用户名] \
         -e CAIXIN_PASSWORD=[你的密码] \
         -e READWISE_TOKEN=[你的token] \
         -v [你本地的某个空文件夹]:/var/app/data fr0der1c/readform:latest
     ```
    `-e` 表示给容器添加环境变量。目前，有以下环境变量可用：
    - `READFORM_WEBSITES`：您要订阅的网站。必填。 允许值：`the_initium`、`caixin`。
    - `READFORM_SAVE_FIRST_FETCH`：是否保存 RSS feed 的第一次提取。 允许值：`yes`、`no`。 默认值为“yes”，这意味着程序运行后将立即将第一批文章保存到 Reader。如果您已经自己将这些网页保存到了 Reader，这会将这些项目添加到您的 library 的顶部。若更改为 no，则在程序启动后发布的新文章才会保存到 Reader。 
    - `THE_INITIUM_USERNAME`：用于登录端的用户名。 如果 `the_initium` 在 `READFORM_WEBSITES` 中，这是必需的，否则是可选的。
    - `THE_INITIUM_PASSWORD`：用于登录端的密码。 如果 `the_initium` 在 `READFORM_WEBSITES` 中，这是必需的，否则是可选的。
    - `CAIXIN_USERNAME`：用于登录财新的用户名。 如果 `caixin` 在 `READFORM_WEBSITES` 中，这是必需的，否则是可选的。
    - `CAIXIN_PASSWORD`：用于登录财新的密码。 如果 `caixin` 在 `READFORM_WEBSITES` 中，这是必需的，否则是可选的。
    - `READWISE_TOKEN`: 您的 Readwise 的 token，[可在此处获取](https://readwise.io/access_token)。
    - `READWISE_READER_LOCATION`：文章保存的位置。可选，默认为 `feed`。 有效值如下：`new`、`later`、`archive` 或 `feed`。
   
    `-v` 参数将本地文件夹绑定到容器中的数据存储文件夹，这是持久化数据所必需的，例如程序需要知道哪些文章已经保存到 Reader 过了，因此不用再次提交。但是，如果您只想快速测试本项目的功能，则可以省略这部分。该路径必须为绝对路径，例如，在 macOS 下可能为 `/Users/fr0der1c/readform_data`。
2. 一切就绪。新文章将出现在您的 Reader 的 feed 流中。如果它并没有如期工作，您可以使用 `docker logs [container-id]` 命令检查日志，因为本项目仍处于萌芽阶段，并且可能包含 bug。 如果您看到任何异常日志/崩溃/文章不全的情况，请随时通过 GitHub issue 反馈！

## FAQ
### 使用此程序需要订阅吗？
这个项目是完全免费和开源的。 但是，**您需要成为网站的订户才能获得完整的文章内容**。 我们不直接提供帐户或完整的文章内容，因为媒体网站和作者都需要获得经济支持以继续前进。

### 这个项目可以绕过付费墙吗？
不能。这个项目的目的是为了改善专业阅读者的工作流程，而不是打破付费墙。您需要使用自己的用户名、密码登录网站。

### 交出我的密码安全吗？
是的。该程序在您的计算机上运行，您的密码将始终保留在本地。我们没有服务器，也不收集用户名和密码。该程序仅对您订阅的站点进行必要的网络请求，并且您的密码将仅在这些站点上使用。

### 如果 Readform 当前不支持我想要的网站怎么办？
我并无计划支持所有付费网站，因为这是一个旨在改善我个人的阅读工作流程的项目。 但是，该项目提供了易于使用的接口，因此您可以轻松添加网站 `Agent`。 欢迎提交 Pull Request！如果您没有代码能力，也可以在 Issue 中提出请求，或许我们（或者社区中的其他成员）会进行跟进。

## 免责声明
使用此代码，即视为您同意以下声明：

本项目**仅供个人使用**，旨在提供更好的阅读体验。 它仅代表用户自动执行操作，**不会以任何方式打破网站所有者设置的任何限制**。 您应自行承担使用本项目的风险。