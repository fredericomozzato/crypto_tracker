class HoldingsController < ApplicationController
  before_action :set_portfolio, only: %i[new create]

  def new
    @holding = @portfolio.holdings.build
    @coins = Coin.all.order(:ticker)
  end

  def create
    @holding = @portfolio.holdings.build holding_params
    if @holding.save
      redirect_to @portfolio, notice: t('.success', ticker: @holding.ticker)
    else
      flash.now[:alert] = t '.fail'
    end
  end

  private

  def set_portfolio
    @portfolio = Portfolio.find params[:portfolio_id]
  end

  def holding_params
    params.require(:holding).permit(:coin_id, :amount, :portfolio_id)
  end
end
