Rails.application.routes.draw do
  devise_for :users

  get 'up' => 'rails/health#show', as: :rails_health_check

  root 'portfolios#index'

  resources :portfolios, only: %i[index new create show destroy] do
    resources :holdings, only: %i[new create]
  end

  resources :holdings, only: %i[edit update destroy]
end
