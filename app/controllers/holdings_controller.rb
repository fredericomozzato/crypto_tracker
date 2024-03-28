class HoldingsController < ApplicationController
  before_action :set_portfolio, :set_coins, only: %i[new create]

  def new
    @holding = @portfolio.holdings.build
  end

  def create
    @holding = @portfolio.holdings.build holding_params
    if @holding.save
      redirect_to @portfolio, notice: t('.success', ticker: @holding.ticker)
    else
      flash.now[:alert] = t '.fail'
      render :new, status: :unprocessable_entity
    end
  end

  def update
    @holding = Holding.find(params.dig(:holding, :id))
    @holding.deposit BigDecimal(params.dig(:holding, :amount))
    redirect_to @holding.portfolio,
                notice: t('.success',
                          amount: params.dig(:holding, :amount),
                          ticker: @holding.ticker)
  end

  private

  def set_portfolio
    @portfolio = Portfolio.find params[:portfolio_id]
  end

  def set_coins
    @coins = Coin.all.order(:ticker)
  end

  def holding_params
    params.require(:holding).permit(:coin_id, :amount, :portfolio_id)
  end
end
