# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[7.1].define(version: 2024_03_23_180043) do
  # These are extensions that must be enabled in order to support this database
  enable_extension "plpgsql"

  create_table "accounts", force: :cascade do |t|
    t.bigint "owner_id", null: false
    t.string "uuid", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["owner_id"], name: "index_accounts_on_owner_id"
    t.index ["owner_id"], name: "unique_account_owner", unique: true
    t.index ["uuid"], name: "unique_account_uuid", unique: true
  end

  create_table "coins", force: :cascade do |t|
    t.string "name", null: false
    t.string "api_id", null: false
    t.string "ticker", null: false
    t.string "icon", null: false
    t.decimal "rate", precision: 16, scale: 8, default: "0.0"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["api_id"], name: "unique_coin_api_id", unique: true
    t.index ["icon"], name: "unique_coin_icon", unique: true
    t.index ["name"], name: "unique_coin_name", unique: true
    t.index ["ticker"], name: "unique_coin_ticker", unique: true
  end

  create_table "holdings", force: :cascade do |t|
    t.bigint "portfolio_id", null: false
    t.bigint "coin_id", null: false
    t.decimal "amount", precision: 16, scale: 8, default: "0.0"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["coin_id"], name: "index_holdings_on_coin_id"
    t.index ["portfolio_id", "coin_id"], name: "portfolio_coin_id", unique: true
    t.index ["portfolio_id"], name: "index_holdings_on_portfolio_id"
  end

  create_table "portfolios", force: :cascade do |t|
    t.bigint "account_id", null: false
    t.string "name", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["account_id"], name: "index_portfolios_on_account_id"
  end

  create_table "users", force: :cascade do |t|
    t.string "email", default: "", null: false
    t.string "encrypted_password", default: "", null: false
    t.string "reset_password_token"
    t.datetime "reset_password_sent_at"
    t.datetime "remember_created_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["email"], name: "index_users_on_email", unique: true
    t.index ["reset_password_token"], name: "index_users_on_reset_password_token", unique: true
  end

  add_foreign_key "accounts", "users", column: "owner_id"
  add_foreign_key "holdings", "coins"
  add_foreign_key "holdings", "portfolios"
  add_foreign_key "portfolios", "accounts"
end
