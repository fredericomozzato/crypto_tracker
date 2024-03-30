class ApplicationController < ActionController::Base
  before_action :authenticate_user!

  protected

  def authorize_owner(portfolio)
    redirect_to root_path, alert: t(:unauthorized) unless portfolio.owner == current_user
  end
end
