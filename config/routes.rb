Rails.application.routes.draw do
  devise_for :users

  get 'up' => 'rails/health#show', as: :rails_health_check

  root 'dashboard#show'

  get '/dashboard', to: 'dashboard#show'

  resources :account, only: %i[show]

  resources :portfolios, only: %i[index new create show edit destroy] do
    resources :holdings, only: %i[new create]
  end

  resources :holdings, only: %i[edit update destroy]
end
