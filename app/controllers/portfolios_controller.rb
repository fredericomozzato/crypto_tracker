class PortfoliosController < ApplicationController
  before_action :set_account, only: %i[create]

  def create
    @portfolio = @account.portfolios.build(portfolio_params)

    if @portfolio.save
      redirect_to portfolios_path, notice: t('.success')
    else
      flash.now[:alert] = t '.fail'
    end
  end

  private

  def set_account
    @account = current_user.account
  end

  def portfolio_params
    params.require(:portfolio).permit(:name)
  end
end
