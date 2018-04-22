package core

import (
	"testing"
	"time"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTxPoolImpl(t *testing.T) {
	Convey("Test of TxPool", t, func() {
		txp := TxPoolImpl{}
		tx := Tx{
			Version: 1,
			Time:    time.Now().Unix(),
		}
		Convey("Add", func() {
			txp.Add(tx)
			So(len(txp.txMap), ShouldEqual, 1)
		})
		Convey("Del", func() {
			txp.Del(tx)
			So(len(txp.txMap), ShouldEqual, 0)
		})

		Convey("Find", func() {
			txp.Add(tx)
			tx2, err := txp.Find(tx.Hash())
			So(err, ShouldBeNil)
			So(tx2.Time, ShouldEqual, tx.Time)

			_, err = txp.Find([]byte("hello"))
			So(err, ShouldNotBeNil)
		})
	})
}