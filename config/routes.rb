Rails.application.routes.draw do
  devise_for :users

  get 'up' => 'rails/health#show', as: :rails_health_check

  root 'pages#index'

  resources :portfolios, only: %i[index create show destroy]
end
