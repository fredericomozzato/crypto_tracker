require 'rails_helper'

RSpec.describe RefreshRatesJob, type: :job do
  describe '#perform' do
    it 'Calls CoinService#refresh_rates' do
      coin_spy = spy CoinService
      stub_const 'CoinService', coin_spy

      RefreshRatesJob.new.perform

      expect(coin_spy).to have_received(:refresh_rates).once
    end
  end
end
