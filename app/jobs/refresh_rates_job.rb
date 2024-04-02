class RefreshRatesJob
  include Sidekiq::Job

  def perform
    CoinService.refresh_rates
  end
end
