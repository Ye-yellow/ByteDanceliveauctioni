UPDATE user_orders AS orders
JOIN auction_users AS users ON users.id = orders.main_account_id
SET orders.shop_name = COALESCE(NULLIF(users.nickname, ''), NULLIF(users.username, ''), orders.shop_name)
WHERE orders.source = 'auction'
  AND (orders.shop_name = '' OR orders.shop_name = '直播竞拍');
