# Crypto Tracker

Crypto currency portfolio manager built with Ruby on Rails.

**Author: Frederico Mozzato**

## Project overview

I've always wanted a simple tool to manage my crypto currency portfolio, so I decided to make my own! 

Crypto Tracker allows users to create accounts and multiple portfolios to track crypto assets. The application uses the CoinGecko API underneath to offer near real time rates, while supporting the top 100 coins by market cap.

This app is currently optimised to be used in a desktop's browser. Maybe in the future I'll add mobile compatibility, but for now, it may not work properly in very small screens.

## Project setup

The application uses PostgreSQL database. The db is set up in a Docker container configured in the `./compose.yaml` file. The credentials are setup using a `.env` file in the application's root folder with the following template:

```
POSTGRES_USER=<your_user>
POSTGRES_PASSWORD=<your_password>
```

Install all the gems with

```
$ bundle install
```

To run the application you execute

```
$ bin/dev
```

**IMPORTANT:** when running the application for the first time you should also run `rails db:prepare` after the application is up. This will ensure that all the tables are created and the migrations are applied.

## Running tests
