FactoryBot.define do
  factory :portfolio do
    account { create(:user).account }
    sequence(:name) { "Portfolio #{_1}" }
  end
end
