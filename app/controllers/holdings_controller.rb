class HoldingsController < ApplicationController
  before_action :set_holding,                      only: %i[edit update destroy]
  before_action :set_portfolio, :set_coins,        only: %i[new create update destroy]
  before_action -> { authorize_owner @portfolio }, only: %i[create update destroy]

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

  def edit; end

  def update
    forward_operation

    @holding.save
    redirect_to @holding.portfolio,
                notice: t(".success_#{params.dig(:holding, :operation)}",
                          amount: params.dig(:holding, :amount),
                          ticker: @holding.ticker)
  end

  def destroy
    @holding.destroy
    redirect_to @holding.portfolio,
                notice: t('.success', ticker: @holding.ticker)
  end

  private

  def set_portfolio
    @portfolio = Portfolio.find_by(id: params[:portfolio_id]) || @holding.portfolio
  end

  def set_coins
    @coins = Coin.all.order(:ticker)
  end

  def set_holding
    @holding = Holding.find params[:id]
  end

  def holding_params
    params.require(:holding).permit(:coin_id, :amount, :portfolio_id)
  end

  def forward_operation
    case params.dig(:holding, :operation)
    in 'deposit'  then @holding.deposit  BigDecimal(params.dig(:holding, :amount))
    in 'withdraw' then @holding.withdraw BigDecimal(params.dig(:holding, :amount))
    in 'update'   then @holding.amount = BigDecimal(params.dig(:holding, :amount))
    end
  end
end
