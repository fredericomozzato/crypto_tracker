class RefreshRatesJob < ApplicationJob
  def perform
    CoinService.refresh_rates
  end
end
