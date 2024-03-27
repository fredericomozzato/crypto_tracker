class PortfoliosController < ApplicationController
  before_action :set_account, only: %i[create index]

  def index
    @portfolios = @account.portfolios
  end

  def create
    @portfolio = @account.portfolios.build(portfolio_params)

    if @portfolio.save
      redirect_to @portfolio, notice: t('.success')
    else
      flash.now[:alert] = t '.fail'
    end
  end

  def show
    @portfolio = Portfolio.includes(:holdings).find params[:id]
    @holdings = @portfolio.holdings.sort_by(&:value).reverse
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
