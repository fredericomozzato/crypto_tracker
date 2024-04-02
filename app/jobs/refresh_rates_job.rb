class RefreshRatesJob < ApplicationJob
  queue_as :default

  def perform
    CoinService.refresh_rates
  end
end
