#!/bin/bash

mockgen -source=./dao/order_dao.go -destination=dao/mocks/order_dao_mock.go -package=mocks
mockgen -source=./dao/order_product_dao.go -destination=dao/mocks/order_product_dao_mock.go -package=mocks
mockgen -source=./dao/order_log_dao.go -destination=dao/mocks/order_log_dao_mock.go -package=mocks

echo "Mocks generated successfully."