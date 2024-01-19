# Readform
[Chinese version](./README_ZH.md)

**Important**: Readform 1.0.0 is now released with significant updates. This version has **completely rewritten** the entire codebase in Go language, replaced selenium with chromedp, and optimized database performance. We anticipate that this refactoring will make the program **more robust and reliable**. As some configuration fields have been adjusted and the database has been optimized, if you have been using Readform before version 1.0.0, we suggest you **delete the data directory** when upgrading and start using Readform from scratch.


---------

This program sends **full article content** of paywalled news websites to your [Readwise Reader](https://readwise.io/read) feed to help you get a unified reading workflow. RSS feed output may be supported in the future.

Currently supported websites:
- The initium (端传媒)
- Caixin (财新)
- Financial Times (FT)

Will be supported in the future:
- WSJ（Wall street journal)
- FTChinese（FT中文网）

## Why I built Readform
There is plenty of high-quality subscription-based media on the market. And I respect their work. Being said that, I believe it's the subscriber's right to read in different forms he/she likes, e.g. RSS reader. **Pro readers have their own customized reading workflow. "Pro" media that charges for their articles should respect their reader's own choice.** Since there is no official full-article RSS feed support for these websites, I decided to make my own.

Currently, I use Readwise Reader as my RSS reader, so I made the Readwise Reader output integration. However, RSS output may also be supported in the future.

The final goal of this project is to push these websites to provide their official full-content RSS feed for their subscribers. Before that, let's use this program!

## How it works
The program gets the latest articles using the website's RSS feed continuously. When there are new articles, it simulates a browser(using Firefox and Selenium) and logs in using your credentials to get full HTML content. Lazy-loading images will be properly handled, so you don't have to worry about missing images. The program will send the article URL and its HTML content to Readwise Reader using the official Reader API, so you can see them in your feed section.


## Quick start
Readform is not a cloud-based service. Instead, you need to run it on your own machine. This brings you maximum safety since your username and password are required for using this program. You can install Readform on a local device(PC, Mac, NAS, Raspberry Pi, ...) or deploy it on a VPS.

Running in Docker is the recommended way to use Readform. If you don't have Docker on your computer, you can [download it here](https://docs.docker.com/get-docker/).

1. Run this program in Docker using the command below in the terminal:
    ```commandline
    docker run --restart=always -d -p 5000:5000\
        -v [your-local-empty-path]:/var/app/data fr0der1c/readform:latest
    ```
   The `-v` parameter binds a local directory to the app data directory in the container. This is necessary to persist data and states, such as articles saved to Reader. However, if you only want to have a quick test of functionalities, it's ok to omit this part.
2. Visit http://localhost:5000 and configure Readform via web UI.
   ![Readform screenshot](./screenshot.png)
3. You're all set. New articles will appear in your Reader feed section. You can check the logs using `docker logs [container-id]` command if it didn't work well, since this is a rather new project and may contain bugs. If you see any abnormal logs/crashes/partial content, feel free to submit an issue!

## FAQ
### Is a subscription required for using this program?
This project is completely free and open source. However, **you do need to be a subscriber of a website to get full article content**. We do not directly provide accounts or full article content because it's important for the press and authors to be financially supported to keep going.

### Does this project bypasses paywall?
No. This project aims to improve workflow for professional readers instead of breaking paywalls. You need to use your own credentials to log in to websites.

### Is it safe to hand over my password?
Yes. The program runs on your computer and your password will always stay local. We have no servers and do not collect usernames and passwords. The program only does necessary network requests to the sites you subscribed to, and your password will only be used on these sites.

### What if the website I wanted is not supported?
I'm not planning to support all paywalled websites since this is a project that is intended to improve my reading workflow. However, the project provides easy-to-use interfaces, so you can add a website `Agent` easily. Pull requests are welcomed!

## Disclaimer
By using this code, you are considered to agree with the following statements:

This project is **only for personal use**, aiming to provide better reading experience. It only automates user actions on user's behalf and **does not break any limit set by website's owner in any way**. You should use at your own risk. 