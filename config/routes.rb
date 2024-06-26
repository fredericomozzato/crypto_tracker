require 'sidekiq/web'
require 'sidekiq/cron/web'

Rails.application.routes.draw do
  devise_for :users

  mount Sidekiq::Web => '/sidekiq'
  get 'up' => 'rails/health#show', as: :rails_health_check

  root 'dashboard#show'

  get '/dashboard', to: 'dashboard#show'

  resources :account, only: %i[show]

  resources :portfolios do
    resources :holdings, only: %i[new create]
  end

  resources :holdings, only: %i[edit update destroy]
end
