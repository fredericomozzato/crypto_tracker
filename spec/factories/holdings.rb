FactoryBot.define do
  factory :holding do
    portfolio
    coin
    sequence(:amount) { "#{_1}.#{_1}#{_1}" }
  end
end
