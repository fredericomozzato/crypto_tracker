# Crypto Tracker

Crypto currency portfolio manager built with Ruby on Rails.

**Author: Frederico Mozzato**

## Project overview

I've always wanted a simple tool to manage my crypto currency portfolio, so I decided to make my own! 

Crypto Tracker allows users to create accounts and multiple portfolios to track crypto assets. The application uses the [CoinGecko API](https://www.coingecko.com/en/api) underneath to offer near real time rates, while supporting the top 100 coins by market cap.

## Project setup

The application uses PostgreSQL database. The db is set up in a Docker container configured in the `./compose.yaml` file. The credentials are setup using a `.env` file in the application's root folder with the following template:

```
POSTGRES_USER=<your_user>
POSTGRES_PASSWORD=<your_password>
```

Next, install all the gems with

```
$ bundle install
```

To run the application you execute

```
$ bin/dev
```

**IMPORTANT:** before starting the application for the first time you should run the services with compose and then `rails db:prepare` after the db service is up. This will ensure that all the tables are created and the migrations are applied.

The compose file will also run a Redis service for the Sidekiq worker that runs scheduled jobs, refreshing coins' rates every minute. To reduce the burden on systems running the application the Sidekiq worker was not added to a separate container, but is executed locally with the Procfile.

### Seeds

The application uses seeds. There's a single user that you can try out with the following credentials for login:

- Email: user@email.com
- Password: 123456

The seeds file will also import the top 100 coins by market cap from the Coin Gecko API, making them available to use in the application.

### Importing coins

There's a rake task built for admin purposes that can be run to manually import coins:

```
$ rake coins:import
```

This task will run the same service for importing coins as the seeds file do. If there are new coins in the top 100 by market cap it'll import them. It won't remove any coin and it won't duplicate existing or overwrite existing coins. It only imports new coins that are not yet present in the database.


## Running tests

The tests run locally, but make sure that all services are up with with the `docker compose` command. With the database service up you can execute `rspec` from the application's root to run the test suite.