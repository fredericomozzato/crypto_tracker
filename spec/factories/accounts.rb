FactoryBot.define do
  factory :account do
    owner
    uuid { 'test-uuid' }
  end
end
