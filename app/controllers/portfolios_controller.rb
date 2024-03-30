class PortfoliosController < ApplicationController
  before_action :set_account,                      only: %i[index create]
  before_action :set_portfolio,                    only: %i[show destroy]
  before_action -> { authorize_owner @portfolio }, only: %i[show destroy]

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
    @holdings = @portfolio.holdings.sort_by(&:value).reverse
  end

  def destroy
    @portfolio.destroy
    redirect_to portfolios_path, notice: t('.success')
  end

  private

  def set_account
    @account = current_user.account
  end

  def set_portfolio
    @portfolio = Portfolio.includes(:holdings).find params[:id]
  end

  def portfolio_params
    params.require(:portfolio).permit(:name)
  end
end
