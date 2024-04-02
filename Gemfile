source 'https://rubygems.org'

ruby '3.2.2'

gem 'dotenv', groups: %i[development test]

gem 'bootsnap', require: false
gem 'cssbundling-rails'
gem 'devise', '~> 4.9'
gem 'faraday', '~> 2.9'
gem 'foreman', '~> 0.87.2'
gem 'jbuilder'
gem 'jsbundling-rails'
gem 'pg', '~> 1.1'
gem 'puma', '>= 5.0'
gem 'rails', '~> 7.1.3', '>= 7.1.3.2'
gem 'sidekiq', '~> 7.2'
gem 'sidekiq-cron', '~> 1.12'
gem 'sprockets-rails'
gem 'stimulus-rails'
gem 'turbo-rails'
gem 'tzinfo-data', platforms: %i[windows jruby]

group :development, :test do
  gem 'debug', platforms: %i[mri windows]
  gem 'factory_bot_rails'
  gem 'rubocop'
  gem 'rubocop-rails'
end

group :development do
  # Use console on exceptions pages [https://github.com/rails/web-console]
  gem 'web-console'
end

group :test do
  gem 'capybara'
  gem 'cuprite'
  gem 'rspec-rails'
  gem 'simplecov', '~> 0.22.0', require: false
end
