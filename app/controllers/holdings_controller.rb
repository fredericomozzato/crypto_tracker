class HoldingsController < ApplicationController
  def create
    @portfolio = Portfolio.find params[:portfolio_id]
    @holding = @portfolio.holdings.build holding_params
    @holding.save
  end

  private

  def holding_params
    params.require(:holding).permit(:coin_id, :amount, :portfolio_id)
  end
end
