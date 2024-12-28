package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tidwall/buntdb"
)

type UserTokens struct {
	VercelToken      string `json:"vercel_token"`
	CloudflareToken  string `json:"cloudflare_token"`
	CloudflareZoneID string `json:"cloudflare_zone_id"`
	VercelProjectID  string `json:"vercel_project_id"`
}

var (
	db                   *buntdb.DB
	userAutoRedirectLock sync.Mutex
	userAutoRedirectMap  map[int64]chan bool
	secretCode           string
)

func init() {
	var err error
	db, err = buntdb.Open("user_tokens.db")
	if err != nil {
		panic(err)
	}
	userAutoRedirectMap = make(map[int64]chan bool)

}

func getUserTokens(userID int64) (UserTokens, error) {
	var tokens UserTokens
	err := db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(fmt.Sprintf("user:%d", userID))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &tokens)
	})
	return tokens, err
}

func setUserTokens(userID int64, tokens UserTokens) error {
	jsonTokens, err := json.Marshal(tokens)
	if err != nil {
		return err
	}
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(fmt.Sprintf("user:%d", userID), string(jsonTokens), nil)
		return err
	})
}
func checkAllTokensPresent(tokens UserTokens) bool {
	return tokens.VercelToken != "" && tokens.CloudflareToken != "" && tokens.CloudflareZoneID != "" && tokens.VercelProjectID != ""
}

func handleTelegramCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	userID := update.Message.From.ID
	username := update.Message.From.UserName

	logInfo(userID, username, fmt.Sprintf("Received command: %s", update.Message.Command()))


	tokens, _ := getUserTokens(int64(userID))

	if !isWhitelisted(int64(userID)) && !isAdminCommand(update.Message.Command()) {
		msg.Text = "🚫 You are not authorized to use this bot. Please contact an admin for access."
		bot.Send(msg)
		return
	}

	if !checkAllTokensPresent(tokens) && !isTokenSetupCommand(update.Message.Command()) {
		msg.Text = "🔑 Please ensure all your API tokens are set using the appropriate commands. Use /help to set the tokens."
		bot.Send(msg)
		return
	}

	switch update.Message.Command() {
	case "start":
		msg.Text = "A Telegram bot that integrates with Vercel and Cloudflare to manage domains and redirect rules. This bot allows users to add domains to Vercel, set up redirects in Cloudflare, and perform various administrative tasks through Telegram commands. \n Use /help to view the commands"
	case "setverceltoken":
		token := update.Message.CommandArguments()
		if token == "" {
			msg.Text = "🚫 Vercel token cannot be empty. Please provide a valid token."
			bot.Send(msg)
			return
		}
		tokens.VercelToken = token
		err := setUserTokens(int64(userID), tokens)
		if err != nil {
			msg.Text = "❌ Error saving Vercel token: " + err.Error()
		} else {
			msg.Text = "✅ Vercel token saved successfully!"
		}

	case "setcloudflaretoken":
		token := update.Message.CommandArguments()
		if token == "" {
			msg.Text = "🚫 Cloudflare token cannot be empty. Please provide a valid token."
			bot.Send(msg)
			return
		}
		tokens.CloudflareToken = token
		err := setUserTokens(int64(userID), tokens)
		if err != nil {
			msg.Text = "❌ Error saving Cloudflare token: " + err.Error()
		} else {
			msg.Text = "✅ Cloudflare token saved successfully!"
		}

	case "setcloudflarezoneid":
		zoneID := update.Message.CommandArguments()
		if zoneID == "" {
			msg.Text = "🚫 Cloudflare Zone ID cannot be empty. Please provide a valid Zone ID."
			bot.Send(msg)
			return
		}
		tokens.CloudflareZoneID = zoneID
		err := setUserTokens(int64(userID), tokens)
		if err != nil {
			msg.Text = "❌ Error saving Cloudflare Zone ID: " + err.Error()
		} else {
			msg.Text = "✅ Cloudflare Zone ID saved successfully!"
		}

	case "setvercelprojectid":
		projectID := update.Message.CommandArguments()
		if projectID == "" {
			msg.Text = "🚫 Vercel Project ID cannot be empty. Please provide a valid Project ID."
			bot.Send(msg)
			return
		}
		tokens.VercelProjectID = projectID
		err := setUserTokens(int64(userID), tokens)
		if err != nil {
			msg.Text = "❌ Error saving Vercel Project ID: " + err.Error()
		} else {
			msg.Text = "✅ Vercel Project ID saved successfully!"
		}

	case "gettokens":
		msg.Text = fmt.Sprintf(
			"🔑 Your tokens:\nVercel Token: %s\nCloudflare Token: %s\nCloudflare Zone ID: %s\nVercel Project ID: %s",
			tokens.VercelToken, tokens.CloudflareToken, tokens.CloudflareZoneID, tokens.VercelProjectID,
		)
	case "settokens":
		args := update.Message.CommandArguments()
		var newTokens UserTokens
		err := json.Unmarshal([]byte(args), &newTokens)
		if err != nil {
			msg.Text = "🚫 Invalid JSON format. Please provide tokens in the format: {\"vercel_token\":\"...\",\"cloudflare_token\":\"...\",\"cloudflare_zone_id\":\"...\",\"vercel_project_id\":\"...\"}"
		} else {
			err = setUserTokens(int64(userID), newTokens)
			if err != nil {
				msg.Text = "❌ Error saving tokens: " + err.Error()
			} else {
				msg.Text = "✅ Tokens saved successfully!"
			}
		}

	case "getdomains":
		domains, err := getVercelDomains(tokens.VercelProjectID, tokens.VercelToken)
		if err != nil {
			msg.Text = "❌ Error getting domains: " + err.Error()
		} else {
			msg.Text = "🌐 Current domains:\n" + strings.Join(domains, "\n")
		}

	case "setdomain":
		args := update.Message.CommandArguments()
		if args == "" {
			msg.Text = "🚫 Please provide a domain name. Usage: /setdomain your-domain.vercel.app"
		} else {
			err := addVercelDomain(tokens.VercelProjectID, args, tokens.VercelToken)
			if err != nil {
				msg.Text = "❌ Error adding domain: " + err.Error()
			} else {
				msg.Text = "✅ Domain added successfully: " + args
			}
		}

	case "deletedomain":
		args := update.Message.CommandArguments()
		if args == "" {
			msg.Text = "🚫 Please provide a domain name. Usage: /deletedomain your-domain.vercel.app"
		} else {
			err := deleteVercelDomain(tokens.VercelProjectID, args, tokens.VercelToken)
			if err != nil {
				msg.Text = "❌ Error deleting domain: " + err.Error()
			} else {
				msg.Text = "✅ Domain deleted successfully: " + args
			}
		}

	case "getredirects":
		rules, err := getCloudflareRedirectRules(tokens.CloudflareZoneID, tokens.CloudflareToken)
		if err != nil {
			msg.Text = "❌ Error fetching redirect rules: " + err.Error()
		} else {
			var rulesText strings.Builder
			rulesText.WriteString("🔄 Current redirect rules:\n\n")
			for _, rule := range rules {
				rulesText.WriteString(fmt.Sprintf("📝 Description: %s\n", rule.Description))
				rulesText.WriteString(fmt.Sprintf("🔍 Expression: %s\n", rule.Expression))
				rulesText.WriteString(fmt.Sprintf("🌐 Target URL: %s\n", rule.ActionParameters.FromValue.TargetURL.Value))
				rulesText.WriteString(fmt.Sprintf("🔢 Status Code: %d\n", rule.ActionParameters.FromValue.StatusCode))
				rulesText.WriteString("\n")
			}
			msg.Text = rulesText.String()
		}

	case "setredirect":
		targetURL := update.Message.CommandArguments()
		if targetURL == "" {
			msg.Text = "🚫 Please provide a target URL. Usage: /setredirect https://target-domain.com"
		} else {
			if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
				targetURL = "https://" + targetURL
			}

			parsedURL, err := url.Parse(targetURL)
			if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || !strings.Contains(parsedURL.Host, ".") {
				msg.Text = "🚫 Invalid URL. Please provide a valid target URL."
				return
			}

			vercelDomains, err := getVercelDomains(tokens.VercelProjectID, tokens.VercelToken)
			if err != nil {
				msg.Text = "❌ Error getting Vercel domains: " + err.Error()
			} else {
				found := false
				for _, domain := range vercelDomains {
					if domain == parsedURL.Host {
						found = true
						break
					}
				}
				if !found {
					warningMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("⚠️ Warning: The domain %s is not found in Vercel domains.", parsedURL.Host))
					bot.Send(warningMsg)
				}
			}

			err = setRedirect(targetURL, tokens.CloudflareZoneID, tokens.CloudflareToken)
			if err != nil {
				msg.Text = "❌ Error setting redirect: " + err.Error()
			} else {
				msg.Text = fmt.Sprintf("✅ Redirect rule set successfully. Target URL: %s", targetURL)
			}
		}

	case "startautoredirect":
		userAutoRedirectLock.Lock()
		defer userAutoRedirectLock.Unlock()

		if _, exists := userAutoRedirectMap[int64(userID)]; exists {
			msg.Text = "⏳ Auto-redirect is already running. Use /stopautoredirect to stop it first."
		} else {
			args := strings.Fields(update.Message.CommandArguments())
			if len(args) < 2 {
				msg.Text = "🚫 Please provide a seed text (project name) and refresh time in minutes. Usage: /startautoredirect your-seed-text refresh-time"
			} else {
				seedText := args[0]
				refreshTime, err := strconv.Atoi(args[1])
				if err != nil || refreshTime <= 0 {
					msg.Text = "🚫 Invalid refresh time. Please provide a positive integer for the refresh time in minutes."
				} else {
					stopChan := make(chan bool)
					userAutoRedirectMap[int64(userID)] = stopChan

					go func() {
						defer func() {
							if r := recover(); r != nil {
								errorMsg := fmt.Sprintf("🛑 Unexpected error occurred: %v", r)
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, errorMsg)
								bot.Send(msg)

								// Clean up the auto-redirect
								userAutoRedirectLock.Lock()
								delete(userAutoRedirectMap, int64(userID))
								userAutoRedirectLock.Unlock()

								stopMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛑 Auto-redirect stopped due to an unexpected error. Use /stopautoredirect to clean up if needed.")
								bot.Send(stopMsg)
							}
						}()
						autoRedirectLoop(bot, update.Message.Chat.ID, int64(userID), username, seedText, tokens, refreshTime, stopChan)

						// Clean up after autoRedirectLoop finishes (due to error or stop signal)
						userAutoRedirectLock.Lock()
						delete(userAutoRedirectMap, int64(userID))
						userAutoRedirectLock.Unlock()
					}()

					msg.Text = fmt.Sprintf("🔄 Auto-redirect started. It will update every %d minutes.", refreshTime)
				}
			}
		}

	case "stopautoredirect":
		userAutoRedirectLock.Lock()
		if stopChan, exists := userAutoRedirectMap[int64(userID)]; exists {
			close(stopChan)
			delete(userAutoRedirectMap, int64(userID))
			msg.Text = "⏹️ Auto-redirect stopped."
		} else {
			msg.Text = "🚫 Auto-redirect is not running."
		}
		userAutoRedirectLock.Unlock()

	case "guide":
		fmt.Println("hello")
		guideContent, err := readGuideFile()
		if err != nil {
			msg.Text = "❌ Error reading guide: " + err.Error()
		} else {
			err = sendLongMessage(bot, update.Message.Chat.ID, guideContent)
			if err != nil {
				msg.Text = "❌ Error sending guide: " + err.Error()
			}
		}
		return

	case "whitelistuser":
		args := strings.Fields(update.Message.CommandArguments())
		if len(args) != 2 {
			msg.Text = "🚫 Invalid command. Usage: /whitelistuser <secret_code> <user_id>"
		} else if args[0] != secretCode {
			msg.Text = "🚫 Invalid secret code. Access denied."
		} else {
			userIDToWhitelist, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				msg.Text = "🚫 Invalid user ID. Please provide a valid numeric ID."
			} else {
				err := whitelistUser(userIDToWhitelist)
				if err != nil {
					msg.Text = "❌ Error whitelisting user: " + err.Error()
				} else {
					msg.Text = fmt.Sprintf("✅ User %d has been whitelisted successfully!", userIDToWhitelist)
				}
			}
		}

	case "getallwhitelistedusers":
		args := strings.Fields(update.Message.CommandArguments())
		if len(args) != 1 {
			msg.Text = "🚫 Invalid command. Usage: /getallwhitelistedusers <secret_code>"
		} else if args[0] != secretCode {
			msg.Text = "🚫 Invalid secret code. Access denied."
		} else {
			whitelistedUsers, err := getAllWhitelistedUsers()
			if err != nil {
				msg.Text = "❌ Error retrieving whitelisted users: " + err.Error()
			} else if len(whitelistedUsers) == 0 {
				msg.Text = "📃 There are no whitelisted users."
			} else {
				var userList strings.Builder
				userList.WriteString("📃 Whitelisted Users:\n\n")
				for _, userID := range whitelistedUsers {
					userList.WriteString(fmt.Sprintf("- User ID: %d\n", userID))
				}
				msg.Text = userList.String()
			}
		}

	case "deletewhitelisteduser":
		args := strings.Fields(update.Message.CommandArguments())
		if len(args) != 2 {
			msg.Text = "🚫 Invalid command. Usage: /deletewhitelisteduser <secret_code> <user_id>"
		} else if args[0] != secretCode {
			msg.Text = "🚫 Invalid secret code. Access denied."
		} else {
			userIDToDelete, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				msg.Text = "🚫 Invalid user ID. Please provide a valid numeric ID."
			} else {
				err := deleteWhitelistedUser(userIDToDelete)
				if err != nil {
					msg.Text = "❌ Error deleting whitelisted user: " + err.Error()
				} else {
					msg.Text = fmt.Sprintf("✅ User %d has been removed from the whitelist successfully!", userIDToDelete)
				}
			}
		}

	case "help":
		msg.Text = "📚 Help Menu: \n\n" +
			"API Guide: \n" +
			"/guide - Set up the api tokens and zone id\n\n" +
			"🔑 Token Management: \n" +
			"/setverceltoken <api-token>- Set your Vercel API token\n" +
			"/setcloudflaretoken <api-token>- Set your Cloudflare API token\n" +
			"/setcloudflarezoneid <zone-id>- Set your Cloudflare Zone ID\n" +
			"/setvercelprojectid <project-id>- Set your Vercel Project ID\n" +
			"/gettokens - Display all your API tokens\n\n" +
			"🌐 Domain Management: \n" +
			"/getdomains - Get the list of Vercel domains\n" +
			"/setdomain <domain> - Add a new domain to Vercel\n" +
			"/deletedomain <domain> - Delete a domain from Vercel\n\n" +
			"🔄 Redirects: \n" +
			"/getredirects - Get the list of Cloudflare redirect rules\n" +
			"/setredirect <url> - Set a redirect rule in Cloudflare\n\n" +
			"⏱️ Auto-Redirect: \n" +
			"/startautoredirect <seed-text> <refresh-time> - Start auto-redirect with seed text\n" +
			"/stopautoredirect - Stop auto-redirect"

	case "admin":
		args := strings.Fields(update.Message.CommandArguments())
		if len(args) < 1 {
			msg.Text = "🚫 Invalid command. Usage: /admin <secret_code>"
			bot.Send(msg)
			return
		}
		if args[0] != secretCode {
			msg.Text = "🚫 Invalid secret code. Access denied."
			return
		}
		msg.Text = "🔐 Whitelist Management: \n\n" +
			"/whitelistuser <secret_code> <user_id> - Add a user to the whitelist\n" +
			"/getallwhitelistedusers <secret_code> - Get all whitelisted users\n" +
			"/deletewhitelisteduser <secret_code> <user_id> - Remove a user from the whitelist\n"

	default:
		msg.Text = "❓ Unknown command. Please use /help to get a list of available commands."
	}

	bot.Send(msg)
}

func sendLongMessage(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	const maxLength = 10000

	for len(text) > 0 {
		if len(text) <= maxLength {
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
			fmt.Println("sented1")
			_, err := bot.Send(msg)
			return err
		}

		splitIndex := strings.LastIndex(text[:maxLength], "\n")
		if splitIndex == -1 {
			splitIndex = maxLength
		}

		msg := tgbotapi.NewMessage(chatID, text[:splitIndex])
		msg.ParseMode = "Markdown"
		fmt.Println("sented")

		_, err := bot.Send(msg)
		if err != nil {
			return err
		}

		text = text[splitIndex:]
	}

	return nil
}

func readGuideFile() (string, error) {
	content, err := ioutil.ReadFile("guide.md")
	if err != nil {
		return "", err
	}
	return string(content), nil
}


func isTokenSetupCommand(command string) bool {
	switch command {
	case "setverceltoken", "setcloudflaretoken", "setcloudflarezoneid", "setvercelprojectid", "gettokens", "help", "guide", "admin","whitelistuser", "getallwhitelistedusers", "deletewhitelisteduser":
		return true
	default:
		return false
	}
}

func isAdminCommand(command string) bool {
	switch command {
	case "whitelistuser", "getallwhitelistedusers", "deletewhitelisteduser", "admin", "help","guide":
		return true
	default:
		return false
	}
}