desc 'Import coins from the Coin Gecko API to the database'

namespace :coins do
  task import: :environment do
    puts '=== Starting import ==='
    start = Time.zone.now

    CoinService.import_coins

    finish = Time.zone.now
    puts "=== finished in #{finish - start} ==="
  end
end
