class PortfoliosController < ApplicationController
  before_action :set_account, only: %i[create]

  def index; end

  def create
    @portfolio = @account.portfolios.build(portfolio_params)

    if @portfolio.save
      redirect_to portfolios_path, notice: t('.success')
    else
      flash.now[:alert] = t '.fail'
    end
  end

  def destroy
    @portfolio = Portfolio.find params[:id]
    @portfolio.destroy
    redirect_to portfolios_path, notice: t('.success')
  end

  private

  def set_account
    @account = current_user.account
  end

  def portfolio_params
    params.require(:portfolio).permit(:name)
  end
end
