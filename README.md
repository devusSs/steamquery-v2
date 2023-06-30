# SteamQuery v2 by devusSs

![steam logo](./docs/steamlogo.png)

## Disclaimer

I do not own any of the rights on potential items / skins / pictures / names used in or outside of this program. Every right goes to their respective owner.<br/>
If there is any issues involving copyright, ownership or any related issues (for example the Steam TOS) please contact the owner (devusSs) via [e-mail](mailto:devuscs@gmail.com) or [open an issue here](https://github.com/devusSs/steamquery-v2/issues).<br/>

<b>Please do not use this program if you are unsure about what it does or how you should use it. Please never share your Steam API key with anyone (including the author of this program).</b>

## Why is this program called SteamQuery v2 and how do I use it?

There used to be a version one of this program. That program however had a lot of perhaps unfixable bugs due to the structure of the code.<br/>
Therefor a new version of this program was created focussing on performance and clean code as well as usability.<br/>

This program makes it possible for you to keep track of certain [CSGO](https://www.counter-strike.net/) skins and items in your [Steam](https://steamcommunity.com/) inventory.<br/>
To do that you will need to create a gcloud.json config file on the Google developer page as well as adding the created service account to your editors on the Google sheet.<br/>
You will also need to create a [config file](./files/config.json) for the program to use.<br/>

Please make sure you set up your Google sheet properly as well.<br/>
The empty lines between the item names and corresponding price cells etc. do not matter, the program will ignore them. It simply serves visibility for the user.<br/>

### How to setup your Google sheet to work properly:

![sample table](./docs/table-sample.png)
For that example you would set following variables in your config:

```json
{
  "item_list": {
    "column_letter": "B",
    "start_number": 6,
    "end_number": 28
  },
  "price_column": "J",
  "price_total_column": "H",
  "amount_column": "F",
  "org_cells": {
    "last_updated_cell": "G2",
    "total_value_cell": "F31",
    "error_cell": "M2",
    "difference_cell": "F32"
  },
  "spread_sheet_id": "your spreadsheet id from the URL",
  "steam_api_key": "your api key"
  "steam_user_id_64": 0,
  "watch_dog": {
    "retry_interval": 0,
    "steam_retry_interval": 0,
    "smtp_host": "",
    "smtp_port": 0,
    "smtp_user": "",
    "smtp_password": "",
    "smtp_from": ",
    "smtp_to": ""
  }
}
```

`Retry interval` specifies the integer value in hours how often the program should update the prices / run the query.<br/>
`Steam retry interval` specifies the integer value in minutes how often the program should retry running the query when Steam is down or not working.

To run the program simple execute:

```bash
steamquery-v2
```

This will start the program with default flags. Considering you setup the configs using this guide everything will work fine.<br/>
If you ever need more configuration options simply set them via flags:

```
-l  to set the logging directory
-c  to set the config file path
-g  to set the gcloud config file path
-d  to run the app in debug mode (not needed, simply adds logging overhead)
-du to disable update checks
-v  to print build information
-a  to run the app in analysis mode (checks for potential errors)
-sc to skip checks regarding last updated and error cell on Google Sheets
-b  to enable and run beta features
-w  to run the app in watchdog mode (automatic rerun after specified interval)
```

## Why does this program need my Steam API key and my SteamID64?

This program queries the status of the Steam Sessions Logon and the Steam Community Status for CSGO to check if everything is up and working before running queries against the endpoints.<br/>
These API routes are protected and need a valid Steam API key to work.<br/>
Get your's [here](https://steamcommunity.com/dev/apikey).<br/>

The SteamID64 is needed to query your CSGO inventory (will be introduced in the near future) programmatically.<br/>
This feature will be used to compare your Google sheet with your inventory and add potentially missing items.<br/>
You can get your SteamID64 on different websites, for example [here](https://steamid.uk/). The SteamID64 may be called CommunityID on some sites.<br/>

## Why do I need to count the amounts manually?

While it is technically prossible to query the Steam inventories via the official API and count the items programmatically, it is not possible to look into storage units which many people use to store their cases and capsules.<br/>
As long as that is not possible you will need to count your items manually.

## Why do I need to enter SMTP values and an e-mail?

To run the app manually when wanted you will not need to enter SMTP details. If you do however want to use the watchdog mode (-w flag) you will need to specify SMTP details.<br/>
The app will then send you an e-mail whenever a run fails. This is intended to keep track of your app status when running the app in watchdog mode (for example on a server).

## Why do I need to run the program manually?

You do not! Simply use the `-w` flag on program launch and specify watchdog details (see above) in the config.<br/>

## Problem Solving

You may run the program in analysis mode to check for potential problems.<br/>
To do that execute:

```bash
steamquery-v2 -c "your config path" -g "your gcloud config path" -a -du
```

If that does not help you may open an issue.

## Building and running the app

Either download an already compiled program from the [releases](https://github.com/devusSs/steamquery-v2/releases) section or clone the repository and compile the program yourself. You will need the [Go(lang)](https://go.dev) binaries for that. Use the `Makefile` for more information.

Errors will usually be self-explanatory. Any weird errors may require the use of [Google](https://google.com) or [creating an issue](https://github.com/devusSs/steamquery-v2/issues) on Github.
