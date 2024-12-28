🤖 *Bot Setup Guide*

This guide will assist you in obtaining the necessary credentials from Cloudflare and Vercel to initialize the bot.

*Step 1: Create an API Token on Cloudflare* 🔑

1. Access the Cloudflare Dashboard: [Cloudflare Dashboard](https://dash.cloudflare.com/profile/api-tokens)

2. Create a New Token:
   • Select "Create Token" 🆕
   • Assign a Token Name (e.g., "Bot API Token")

3. Configure Permissions:
   • Under Permissions, choose "Zone" and "Zone.Dynamic Redirect" 🛡️
   • Ensure permission is set to "All Zones"

4. Finalize Token Creation:
   • Select "Continue to summary"
   • Review details and confirm by selecting "Create Token"
   • Securely store the generated token 🔒

📷 [Reference Image 1](https://ibb.co/RH72SBz)
📷 [Reference Image 2](https://ibb.co/SXN09tf)

*Step 2: Obtain Your Cloudflare Zone ID* 🌐

1. Navigate to the [Cloudflare Dashboard](https://dash.cloudflare.com/)

2. Select Your Domain:
   • Click on the appropriate domain name

3. Locate the Zone ID:
   • On the overview page, identify the Zone ID on the right side
   • Copy this identifier for future use 📋

*Step 3: Create an API Token on Vercel* 🗝️

1. Access the [Vercel Dashboard](https://vercel.com/dashboard)

2. Navigate to Account Settings:
   • Select your profile icon, then "Settings"

3. Proceed to Tokens:
   • In the "Tokens" section, select "Add Token"

4. Generate a New Token:
   • Assign a Token Name (e.g., "Bot API Token")
   • Select "Create" and securely store the generated token

📷 [Reference Image 3](https://ibb.co/C247cRn)

*Step 4: Obtain Your Vercel Project ID* 📁

1. Visit the [Vercel Dashboard](https://vercel.com/dashboard)

2. Access Your Project

3. Retrieve the Project ID:
   • Locate and copy the Project ID from the project settings

📷 [Reference Image 4](https://ibb.co/WnDfP8M)

*Step 5: Configure the Bot* ⚙️

Utilize the following commands to configure the bot:

- /setverceltoken : Set Vercel API token
- /setcloudflaretoken : Set Cloudflare API token
- /setcloudflarezoneid : Set Cloudflare Zone ID
- /setvercelprojectid : Set Vercel Project ID

This concludes the setup process. Should you require further assistance, please don't hesitate to inquire.