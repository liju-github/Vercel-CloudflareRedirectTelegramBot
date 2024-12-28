package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tidwall/buntdb"
)

func getVercelDomains(projectId, vercelToken string) ([]string, error) {
	url := fmt.Sprintf("%s/v9/projects/%s/domains", vercelAPIURL, projectId)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+vercelToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get Vercel domains: %s", string(body))
	}

	var result struct {
		Domains []struct {
			Name string `json:"name"`
		} `json:"domains"`
	}
	json.Unmarshal(body, &result)

	domains := make([]string, len(result.Domains))
	for i, domain := range result.Domains {
		domains[i] = domain.Name
	}

	return domains, nil
}

func addVercelDomain(projectId, newDomain, vercelToken string) error {
	url := fmt.Sprintf("%s/v9/projects/%s/domains", vercelAPIURL, projectId)

	payload := map[string]string{
		"name": newDomain,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+vercelToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add Vercel domain: %s", string(body))
	}

	return nil
}

func deleteVercelDomain(projectId, domain, vercelToken string) error {
	domains, err := getVercelDomains(projectId, vercelToken)
	if err != nil {
		return fmt.Errorf("failed to get current domains: %v", err)
	}

	if len(domains) <= 1 {
		return fmt.Errorf("cannot delete the last remaining domain")
	}

	url := fmt.Sprintf("%s/v9/projects/%s/domains/%s", vercelAPIURL, projectId, domain)

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+vercelToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete Vercel domain: %s", string(body))
	}

	return nil
}

func getCloudflareRedirectRules(zoneID, cloudflareToken string) ([]RedirectRule, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/rulesets/phases/http_request_dynamic_redirect/entrypoint", zoneID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+cloudflareToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var redirectRulesResp RedirectRulesResponse
	err = json.Unmarshal(body, &redirectRulesResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return redirectRulesResp.Result.Rules, nil
}

func setRedirect(targetURL, zoneID, cloudflareToken string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/rulesets/phases/http_request_dynamic_redirect/entrypoint", zoneID)

	rule := RedirectRule{
		Action:      "redirect",
		Expression:  "true",
		Description: fmt.Sprintf("Redirect to %s", targetURL),
		Enabled:     true,
		ActionParameters: struct {
			FromValue struct {
				StatusCode int `json:"status_code"`
				TargetURL  struct {
					Value string `json:"value"`
				} `json:"target_url"`
				PreserveQueryString bool `json:"preserve_query_string"`
			} `json:"from_value"`
		}{
			FromValue: struct {
				StatusCode int `json:"status_code"`
				TargetURL  struct {
					Value string `json:"value"`
				} `json:"target_url"`
				PreserveQueryString bool `json:"preserve_query_string"`
			}{
				StatusCode: 301,
				TargetURL: struct {
					Value string `json:"value"`
				}{
					Value: targetURL,
				},
				PreserveQueryString: true,
			},
		},
	}

	payload := map[string]interface{}{
		"rules": []RedirectRule{rule},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+cloudflareToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func autoRedirectLoop(bot *tgbotapi.BotAPI, chatID int64, userID int64, username, seedText string, tokens UserTokens, RedirectRefresh int, stopChan chan bool) {
	ticker := time.NewTicker(time.Duration(RedirectRefresh) * time.Minute)
	defer ticker.Stop()

	tempPreviousDomain := ""

	for {
		newDomain := generateRandomDomain(seedText)

		if tempPreviousDomain != "" {
			if err := deleteVercelDomain(tokens.VercelProjectID, tempPreviousDomain, tokens.VercelToken); err != nil {
				errorMsg := "❌ Error deleting previous domain, make sure your vercel api token and project id is correct \n " + err.Error()
				sendErrorAndStop(bot, chatID, userID, username, errorMsg, err)
				return
			}
		}

		if err := addVercelDomain(tokens.VercelProjectID, newDomain, tokens.VercelToken); err != nil {
			errorMsg := "❌ Error adding new domain, make sure your vercel api token and project id is correct \n " + err.Error()
			sendErrorAndStop(bot, chatID, userID, username, errorMsg, err)
			return
		}

		if err := setRedirect("https://"+newDomain, tokens.CloudflareZoneID, tokens.CloudflareToken); err != nil {
			errorMsg := "❌ Error setting redirect, make sure your cloudflare api token and zone id is correct \n" + err.Error()
			sendErrorAndStop(bot, chatID, userID, username, errorMsg, err)
			return
		}

		currentTime := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
		messageText := fmt.Sprintf("🔄 Auto-redirect updated at %s. New domain: %s. It will update in %d minutes.", currentTime, newDomain, RedirectRefresh)
		msg := tgbotapi.NewMessage(chatID, messageText)
		if _, err := bot.Send(msg); err != nil {
			errorMsg := "❌ Error sending update message"
			sendErrorAndStop(bot, chatID, userID, username, errorMsg, err)
			return
		}
		logInfo(userID, username, fmt.Sprintf("Auto-redirect updated. New domain: %s", newDomain))

		tempPreviousDomain = newDomain

		select {
		case <-ticker.C:
			// Continue to the next iteration
		case <-stopChan:
			return
		}
	}
}

func sendErrorAndStop(bot *tgbotapi.BotAPI, chatID int64, userID int64, username, errorMsg string, err error) {
	msg := tgbotapi.NewMessage(chatID, errorMsg)
	if _, sendErr := bot.Send(msg); sendErr != nil {
		logError(userID, username, "Error sending error message", sendErr)
	}
	logError(userID, username, errorMsg, err)

	stopMsg := tgbotapi.NewMessage(chatID, "🛑 Auto-redirect stopped due to an error. Use /stopautoredirect to clean up.")
	if _, sendErr := bot.Send(stopMsg); sendErr != nil {
		logError(userID, username, "Error sending stop message", sendErr)
	}
}

func generateRandomDomain(seedText string) string {
	rand.Seed(time.Now().UnixNano())
	randomString := fmt.Sprintf("%s-%d", seedText, rand.Intn(10000))
	return fmt.Sprintf("%s.vercel.app", randomString)
}

func whitelistUser(userID int64) error {
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(fmt.Sprintf("whitelist:%d", userID), "true", nil)
		return err
	})
}

func isWhitelisted(userID int64) bool {
	var whitelisted bool
	db.View(func(tx *buntdb.Tx) error {
		_, err := tx.Get(fmt.Sprintf("whitelist:%d", userID))
		whitelisted = (err == nil)
		return nil
	})
	return whitelisted
}

func deleteWhitelistedUser(userID int64) error {
	return db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(fmt.Sprintf("whitelist:%d", userID))
		return err
	})
}

func getAllWhitelistedUsers() ([]int64, error) {
	var whitelistedUsers []int64
	err := db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if strings.HasPrefix(key, "whitelist:") {
				userID, err := strconv.ParseInt(strings.TrimPrefix(key, "whitelist:"), 10, 64)
				if err == nil {
					whitelistedUsers = append(whitelistedUsers, userID)
				}
			}
			return true
		})
	})
	return whitelistedUsers, err
}
