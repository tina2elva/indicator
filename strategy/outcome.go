// Copyright (c) 2021-2024 Onur Cinar.
// The source code is provided under GNU AGPLv3 License.
// https://github.com/cinar/indicator

package strategy

import (
	"math"
	"time"

	"github.com/cinar/indicator/v2/asset"
	"github.com/cinar/indicator/v2/helper"
)

var (
	outcomeFeeConfig CommissionFee = AStockCommissionFee{
		Commission:    0.0003,
		MinCommission: 5.0,
		Slippage:      0.001,
		StampDuty:     0.0005,
	}

	tradeConfig TradeConfig = TradeConfig{
		Balance: 10000.0,
		MinSize: 100.0,
	}
)

type CommissionFee interface {
	Calculate(price float64, size float64, action Action) float64
}

type AStockCommissionFee struct {
	Commission    float64 // commission rate
	MinCommission float64 // minimum commission, if commission is less than this value, it will be set to this value
	Slippage      float64 // slippage rate
	StampDuty     float64 // tax rate 印花税
}

func (f AStockCommissionFee) Calculate(price float64, size float64, action Action) float64 {
	commission := size * price * f.Commission
	if commission < f.MinCommission {
		commission = f.MinCommission
	}

	slippage := size * price * f.Slippage
	stampDuty := 0.0
	if action == Sell {
		stampDuty = size * price * f.StampDuty
	}

	return commission + slippage + stampDuty
}

func SetCommissionFee(fee CommissionFee) {
	outcomeFeeConfig = fee
}

type TradeConfig struct {
	Balance float64 // 初始资金
	MinSize float64 // 最小交易数量
	// StopLoss float64 // 止损点
	// TakeProfit float64 // 止盈点
}

// Outcome simulates the potential result of executing the given actions based on the provided values.
func Outcome1[T helper.Number](values <-chan T, actions <-chan Action) <-chan float64 {
	balance := 1.0
	shares := 0.0

	return helper.Operate(values, actions, func(value T, action Action) float64 {
		if balance > 0 && action == Buy {
			shares = balance / float64(value)
			balance = 0
		} else if shares > 0 && action == Sell {
			balance = shares * float64(value)
			shares = 0
		}

		return balance + (shares * float64(value)) - 1.0
	})
}

// Outcome simulates the potential result of executing the given actions based on the provided values.
func Outcome2[T helper.Number](values <-chan T, actions <-chan Action) <-chan float64 {
	balance := tradeConfig.Balance
	shares := 0.0
	buyCount := 0
	sellCount := 0
	totalFee := 0.0

	return helper.Operate(values, actions, func(value T, action Action) float64 {
		switch action {
		case Buy:
			// 需先卖掉手头的股票，才能买入新的股票
			if shares != 0 {
				break
			}
			// 理论最多可以买多少股
			maxStocks := balance / float64(value)
			// 理论最多可以买多少手
			maxHands := maxStocks / tradeConfig.MinSize
			// 取整，实际可以最多买多少手
			maxHands = math.Floor(maxHands)
			// 实际最多可以买多少股
			maxStocks = maxHands * tradeConfig.MinSize
			// 购买数量
			buyStocks := maxStocks
			// 计算手续费
			for buyStocks >= tradeConfig.MinSize {
				fee := outcomeFeeConfig.Calculate(float64(value), buyStocks, Buy)
				// 实际需要花费的金额
				cost := float64(value)*buyStocks + fee
				// 剩余余额不够买入
				if cost > balance {
					// 买入数量减少
					buyStocks -= tradeConfig.MinSize
				} else {
					balance -= cost
					shares += buyStocks
					buyCount++
					totalFee += fee
					break
				}
			}
		case Sell:
			if shares > 0 {
				fee := outcomeFeeConfig.Calculate(float64(value), shares, Sell)
				cost := float64(value)*shares - fee
				balance += cost
				shares = 0
				sellCount++
				totalFee += fee
			}
		}
		return (balance + (shares * float64(value)) - tradeConfig.Balance) / tradeConfig.Balance
	})
}

// Outcome simulates the potential result of executing the given actions based on the provided values.
func Outcome(values <-chan *asset.Snapshot, actions <-chan Action) <-chan float64 {
	balance := tradeConfig.Balance
	shares := 0.0
	buyCount := 0
	sellCount := 0
	totalFee := 0.0
	lastBuyDate := time.Time{}

	return helper.Operate(values, actions, func(value *asset.Snapshot, action Action) float64 {
		switch action {
		case Buy:
			// 需先卖掉手头的股票，才能买入新的股票
			if shares != 0 {
				break
			}
			// 理论最多可以买多少股
			maxStocks := balance / float64(value.Close)
			// 理论最多可以买多少手
			maxHands := maxStocks / tradeConfig.MinSize
			// 取整，实际可以最多买多少手
			maxHands = math.Floor(maxHands)
			// 实际最多可以买多少股
			maxStocks = maxHands * tradeConfig.MinSize
			// 购买数量
			buyStocks := maxStocks
			// 计算手续费
			for buyStocks >= tradeConfig.MinSize {
				fee := outcomeFeeConfig.Calculate(float64(value.Close), buyStocks, Buy)
				// 实际需要花费的金额
				cost := float64(value.Close)*buyStocks + fee
				// 剩余余额不够买入
				if cost > balance {
					// 买入数量减少
					buyStocks -= tradeConfig.MinSize
				} else {
					balance -= cost
					shares += buyStocks
					buyCount++
					totalFee += fee
					lastBuyDate = value.Date
					break
				}
			}
		case Sell:
			// 当天不能卖出
			if shares > 0 && value.Date.Day() != lastBuyDate.Day() {
				fee := outcomeFeeConfig.Calculate(float64(value.Close), shares, Sell)
				cost := float64(value.Close)*shares - fee
				balance += cost
				shares = 0
				sellCount++
				totalFee += fee
			}
		}
		return (balance + (shares * float64(value.Close)) - tradeConfig.Balance) / tradeConfig.Balance
	})
}
