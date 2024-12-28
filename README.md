

# Vercel-Cloudflare Redirect Bot

A Telegram bot that integrates with Vercel and Cloudflare to manage domains and redirect rules. This bot allows users to add domains to Vercel, set up redirects in Cloudflare, and perform various administrative tasks through Telegram commands.

## Prerequisites

1. **Go**: Ensure you have Go installed on your system. You can download it from the [official Go website](https://golang.org/dl/).
2. **Telegram Bot API Token**: Create a bot on Telegram and get the API token. Follow the [BotFather guide](https://core.telegram.org/bots#botfather) to create a new bot.
3. **Vercel API Token**: Obtain a token from Vercel by creating a personal access token from your [Vercel dashboard](https://vercel.com/account/tokens).
4. **Cloudflare API Token**: Generate a Cloudflare API token with permissions for managing redirect rules. You can create a token from your [Cloudflare dashboard](https://dash.cloudflare.com/profile/api-tokens).

## Setup

### Create a `.env` File

Create a file named `.env` in the root of the project and add the following environment variables:

```
TELEGRAM_TOKEN=your-telegram-bot-token
```

Replace the placeholders with your actual token. 

### Install Dependencies

Ensure you have Go modules enabled and install the required dependencies:

```bash
go mod init vercelredirect
go mod tidy
```

### Run the Bot

To start the bot, use the following command:

```bash
go run main.go 
```
or use 

```bash
go build -o main.go 
```
and then run it by ./main

## Usage

Start a chat with your bot on Telegram and use the following commands:

- `/guide` - Guide to help you get your credentials from vercel and cloudflare
- `/help` - Display available commands and their usage
- `/setverceltoken <token>` - Set your Vercel API token
- `/setcloudflaretoken <token>` - Set your Cloudflare API token
- `/setcloudflarezoneid <zone_id>` - Set your Cloudflare Zone ID
- `/setvercelprojectid <project_id>` - Set your Vercel Project ID
- `/gettokens` - Display all your stored API tokens
- `/getdomains` - List all domains in your Vercel project
- `/setdomain <domain>` - Add a new domain to your Vercel project
- `/deletedomain <domain>` - Delete a domain from your Vercel project
- `/getredirects` - List all redirect rules in Cloudflare
- `/setredirect <url>` - Set up a redirect rule in Cloudflare
- `/startautoredirect <seed> <time>` - Start auto-redirect with the given seed text and time in minutes
- `/stopautoredirect` - Stop the auto-redirect process

Admin Management 
- `/whitelistuser <secret_code> <user_id>` - Add a user to the whitelist,Whitelist yourselves to use the bot.Get USERID from https://t.me/SangMata_BOT using /my command
- `/getallwhitelistedusers <secret_code>` - Get all whitelisted users
- `/deletewhitelisteduser <secret_code> <user_id>` - Remove a user from the whitelist



https://dash.cloudflare.com/profile/api-tokens 
Add a Token Name,Set Permissions as Zone.Dynamic Redirect All Zones 


https://dash.cloudflare.com/
Click your domain and on the overview,get the API zone id mentioned in the right side of the screen 
