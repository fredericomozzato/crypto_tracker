FactoryBot.define do
  factory :user do
    sequence(:email) { "user_#{_1}@email.com" }
    password { '123456' }
  end
end
