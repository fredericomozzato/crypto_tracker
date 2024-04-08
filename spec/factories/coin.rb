FactoryBot.define do
  factory :coin do
    sequence(:name, 'A') { "Coin #{_1}" }
    sequence(:api_id, 'a') { "coin_#{_1}" }
    sequence(:ticker, 'A') { "CN#{_1}" }
    sequence(:icon, 'a') { "coin_#{_1}.jpg" }
    sequence(:rate) { 9.9 + _1 }
    sequence(:price_change) { rand(-100.0..100.0).round 2 }
    active { true }
  end
end
