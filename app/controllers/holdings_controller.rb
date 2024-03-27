class HoldingsController < ApplicationController
  def create
    @portfolio = Portfolio.find params[:portfolio_id]
    @holding = @portfolio.holdings.build holding_params
    if @holding.save
      redirect_to @portfolio, notice: t('.success', ticker: @holding.ticker)
    else
      flash.now[:alert] = t '.fail'
    end
  end

  private

  def holding_params
    params.require(:holding).permit(:coin_id, :amount, :portfolio_id)
  end
end
