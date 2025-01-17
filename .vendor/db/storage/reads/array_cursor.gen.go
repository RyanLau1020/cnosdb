// Generated by tmpl
// https://github.com/benbjohnson/tmpl
//
// DO NOT EDIT!
// Source: array_cursor.gen.go.tmpl

package reads

import (
	"errors"

	"github.com/cnosdatabase/db/tsdb/cursors"
)

const (
	// MaxPointsPerBlock is the maximum number of points in an encoded
	// block in a TSM file. It should match the value in the tsm1
	// package, but we don't want to import it.
	MaxPointsPerBlock = 1000
)

// ********************
// Float Array Cursor

type floatArrayFilterCursor struct {
	cursors.FloatArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.FloatArray
	tmp  *cursors.FloatArray
}

func newFloatFilterArrayCursor(cond expression) *floatArrayFilterCursor {
	return &floatArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewFloatArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.FloatArray{},
	}
}

func (c *floatArrayFilterCursor) reset(cur cursors.FloatArrayCursor) {
	c.FloatArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *floatArrayFilterCursor) Stats() cursors.CursorStats { return c.FloatArrayCursor.Stats() }

func (c *floatArrayFilterCursor) Next() *cursors.FloatArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.FloatArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.FloatArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}
		// Clear buffered timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil
		a = c.FloatArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type floatMultiShardArrayCursor struct {
	cursors.FloatArrayCursor
	cursorContext
	filter *floatArrayFilterCursor
}

func (c *floatMultiShardArrayCursor) reset(cur cursors.FloatArrayCursor, itrs cursors.CursorIterators, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newFloatFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.FloatArrayCursor = cur
	c.itrs = itrs
	c.err = nil
	c.count = 0
}

func (c *floatMultiShardArrayCursor) Err() error { return c.err }

func (c *floatMultiShardArrayCursor) Stats() cursors.CursorStats {
	return c.FloatArrayCursor.Stats()
}

func (c *floatMultiShardArrayCursor) Next() *cursors.FloatArray {
	for {
		a := c.FloatArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		c.count += int64(a.Len())
		if c.count > c.limit {
			diff := c.count - c.limit
			c.count -= diff
			rem := int64(a.Len()) - diff
			a.Timestamps = a.Timestamps[:rem]
			a.Values = a.Values[:rem]
		}
		return a
	}
}

func (c *floatMultiShardArrayCursor) nextArrayCursor() bool {
	if len(c.itrs) == 0 {
		return false
	}

	c.FloatArrayCursor.Close()

	var itr cursors.CursorIterator
	var cur cursors.Cursor
	for cur == nil && len(c.itrs) > 0 {
		itr, c.itrs = c.itrs[0], c.itrs[1:]
		cur, _ = itr.Next(c.ctx, c.req)
	}

	var ok bool
	if cur != nil {
		var next cursors.FloatArrayCursor
		next, ok = cur.(cursors.FloatArrayCursor)
		if !ok {
			cur.Close()
			next = FloatEmptyArrayCursor
			c.itrs = nil
			c.err = errors.New("expected float cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.FloatArrayCursor = next
	} else {
		c.FloatArrayCursor = FloatEmptyArrayCursor
	}

	return ok
}

type floatArraySumCursor struct {
	cursors.FloatArrayCursor
	ts  [1]int64
	vs  [1]float64
	res *cursors.FloatArray
}

func newFloatArraySumCursor(cur cursors.FloatArrayCursor) *floatArraySumCursor {
	return &floatArraySumCursor{
		FloatArrayCursor: cur,
		res:              &cursors.FloatArray{},
	}
}

func (c floatArraySumCursor) Stats() cursors.CursorStats { return c.FloatArrayCursor.Stats() }

func (c floatArraySumCursor) Next() *cursors.FloatArray {
	a := c.FloatArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc float64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.FloatArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerFloatCountArrayCursor struct {
	cursors.FloatArrayCursor
}

func (c *integerFloatCountArrayCursor) Stats() cursors.CursorStats {
	return c.FloatArrayCursor.Stats()
}

func (c *integerFloatCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.FloatArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.FloatArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type floatEmptyArrayCursor struct {
	res cursors.FloatArray
}

var FloatEmptyArrayCursor cursors.FloatArrayCursor = &floatEmptyArrayCursor{}

func (c *floatEmptyArrayCursor) Err() error                 { return nil }
func (c *floatEmptyArrayCursor) Close()                     {}
func (c *floatEmptyArrayCursor) Stats() cursors.CursorStats { return cursors.CursorStats{} }
func (c *floatEmptyArrayCursor) Next() *cursors.FloatArray  { return &c.res }

// ********************
// Integer Array Cursor

type integerArrayFilterCursor struct {
	cursors.IntegerArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.IntegerArray
	tmp  *cursors.IntegerArray
}

func newIntegerFilterArrayCursor(cond expression) *integerArrayFilterCursor {
	return &integerArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewIntegerArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.IntegerArray{},
	}
}

func (c *integerArrayFilterCursor) reset(cur cursors.IntegerArrayCursor) {
	c.IntegerArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *integerArrayFilterCursor) Stats() cursors.CursorStats { return c.IntegerArrayCursor.Stats() }

func (c *integerArrayFilterCursor) Next() *cursors.IntegerArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.IntegerArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.IntegerArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}
		// Clear buffered timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil
		a = c.IntegerArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type integerMultiShardArrayCursor struct {
	cursors.IntegerArrayCursor
	cursorContext
	filter *integerArrayFilterCursor
}

func (c *integerMultiShardArrayCursor) reset(cur cursors.IntegerArrayCursor, itrs cursors.CursorIterators, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newIntegerFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.IntegerArrayCursor = cur
	c.itrs = itrs
	c.err = nil
	c.count = 0
}

func (c *integerMultiShardArrayCursor) Err() error { return c.err }

func (c *integerMultiShardArrayCursor) Stats() cursors.CursorStats {
	return c.IntegerArrayCursor.Stats()
}

func (c *integerMultiShardArrayCursor) Next() *cursors.IntegerArray {
	for {
		a := c.IntegerArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		c.count += int64(a.Len())
		if c.count > c.limit {
			diff := c.count - c.limit
			c.count -= diff
			rem := int64(a.Len()) - diff
			a.Timestamps = a.Timestamps[:rem]
			a.Values = a.Values[:rem]
		}
		return a
	}
}

func (c *integerMultiShardArrayCursor) nextArrayCursor() bool {
	if len(c.itrs) == 0 {
		return false
	}

	c.IntegerArrayCursor.Close()

	var itr cursors.CursorIterator
	var cur cursors.Cursor
	for cur == nil && len(c.itrs) > 0 {
		itr, c.itrs = c.itrs[0], c.itrs[1:]
		cur, _ = itr.Next(c.ctx, c.req)
	}

	var ok bool
	if cur != nil {
		var next cursors.IntegerArrayCursor
		next, ok = cur.(cursors.IntegerArrayCursor)
		if !ok {
			cur.Close()
			next = IntegerEmptyArrayCursor
			c.itrs = nil
			c.err = errors.New("expected integer cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.IntegerArrayCursor = next
	} else {
		c.IntegerArrayCursor = IntegerEmptyArrayCursor
	}

	return ok
}

type integerArraySumCursor struct {
	cursors.IntegerArrayCursor
	ts  [1]int64
	vs  [1]int64
	res *cursors.IntegerArray
}

func newIntegerArraySumCursor(cur cursors.IntegerArrayCursor) *integerArraySumCursor {
	return &integerArraySumCursor{
		IntegerArrayCursor: cur,
		res:                &cursors.IntegerArray{},
	}
}

func (c integerArraySumCursor) Stats() cursors.CursorStats { return c.IntegerArrayCursor.Stats() }

func (c integerArraySumCursor) Next() *cursors.IntegerArray {
	a := c.IntegerArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc int64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.IntegerArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerIntegerCountArrayCursor struct {
	cursors.IntegerArrayCursor
}

func (c *integerIntegerCountArrayCursor) Stats() cursors.CursorStats {
	return c.IntegerArrayCursor.Stats()
}

func (c *integerIntegerCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.IntegerArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.IntegerArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type integerEmptyArrayCursor struct {
	res cursors.IntegerArray
}

var IntegerEmptyArrayCursor cursors.IntegerArrayCursor = &integerEmptyArrayCursor{}

func (c *integerEmptyArrayCursor) Err() error                  { return nil }
func (c *integerEmptyArrayCursor) Close()                      {}
func (c *integerEmptyArrayCursor) Stats() cursors.CursorStats  { return cursors.CursorStats{} }
func (c *integerEmptyArrayCursor) Next() *cursors.IntegerArray { return &c.res }

// ********************
// Unsigned Array Cursor

type unsignedArrayFilterCursor struct {
	cursors.UnsignedArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.UnsignedArray
	tmp  *cursors.UnsignedArray
}

func newUnsignedFilterArrayCursor(cond expression) *unsignedArrayFilterCursor {
	return &unsignedArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewUnsignedArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.UnsignedArray{},
	}
}

func (c *unsignedArrayFilterCursor) reset(cur cursors.UnsignedArrayCursor) {
	c.UnsignedArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *unsignedArrayFilterCursor) Stats() cursors.CursorStats { return c.UnsignedArrayCursor.Stats() }

func (c *unsignedArrayFilterCursor) Next() *cursors.UnsignedArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.UnsignedArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.UnsignedArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}
		// Clear buffered timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil
		a = c.UnsignedArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type unsignedMultiShardArrayCursor struct {
	cursors.UnsignedArrayCursor
	cursorContext
	filter *unsignedArrayFilterCursor
}

func (c *unsignedMultiShardArrayCursor) reset(cur cursors.UnsignedArrayCursor, itrs cursors.CursorIterators, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newUnsignedFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.UnsignedArrayCursor = cur
	c.itrs = itrs
	c.err = nil
	c.count = 0
}

func (c *unsignedMultiShardArrayCursor) Err() error { return c.err }

func (c *unsignedMultiShardArrayCursor) Stats() cursors.CursorStats {
	return c.UnsignedArrayCursor.Stats()
}

func (c *unsignedMultiShardArrayCursor) Next() *cursors.UnsignedArray {
	for {
		a := c.UnsignedArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		c.count += int64(a.Len())
		if c.count > c.limit {
			diff := c.count - c.limit
			c.count -= diff
			rem := int64(a.Len()) - diff
			a.Timestamps = a.Timestamps[:rem]
			a.Values = a.Values[:rem]
		}
		return a
	}
}

func (c *unsignedMultiShardArrayCursor) nextArrayCursor() bool {
	if len(c.itrs) == 0 {
		return false
	}

	c.UnsignedArrayCursor.Close()

	var itr cursors.CursorIterator
	var cur cursors.Cursor
	for cur == nil && len(c.itrs) > 0 {
		itr, c.itrs = c.itrs[0], c.itrs[1:]
		cur, _ = itr.Next(c.ctx, c.req)
	}

	var ok bool
	if cur != nil {
		var next cursors.UnsignedArrayCursor
		next, ok = cur.(cursors.UnsignedArrayCursor)
		if !ok {
			cur.Close()
			next = UnsignedEmptyArrayCursor
			c.itrs = nil
			c.err = errors.New("expected unsigned cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.UnsignedArrayCursor = next
	} else {
		c.UnsignedArrayCursor = UnsignedEmptyArrayCursor
	}

	return ok
}

type unsignedArraySumCursor struct {
	cursors.UnsignedArrayCursor
	ts  [1]int64
	vs  [1]uint64
	res *cursors.UnsignedArray
}

func newUnsignedArraySumCursor(cur cursors.UnsignedArrayCursor) *unsignedArraySumCursor {
	return &unsignedArraySumCursor{
		UnsignedArrayCursor: cur,
		res:                 &cursors.UnsignedArray{},
	}
}

func (c unsignedArraySumCursor) Stats() cursors.CursorStats { return c.UnsignedArrayCursor.Stats() }

func (c unsignedArraySumCursor) Next() *cursors.UnsignedArray {
	a := c.UnsignedArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return a
	}

	ts := a.Timestamps[0]
	var acc uint64

	for {
		for _, v := range a.Values {
			acc += v
		}
		a = c.UnsignedArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			c.ts[0] = ts
			c.vs[0] = acc
			c.res.Timestamps = c.ts[:]
			c.res.Values = c.vs[:]
			return c.res
		}
	}
}

type integerUnsignedCountArrayCursor struct {
	cursors.UnsignedArrayCursor
}

func (c *integerUnsignedCountArrayCursor) Stats() cursors.CursorStats {
	return c.UnsignedArrayCursor.Stats()
}

func (c *integerUnsignedCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.UnsignedArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.UnsignedArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type unsignedEmptyArrayCursor struct {
	res cursors.UnsignedArray
}

var UnsignedEmptyArrayCursor cursors.UnsignedArrayCursor = &unsignedEmptyArrayCursor{}

func (c *unsignedEmptyArrayCursor) Err() error                   { return nil }
func (c *unsignedEmptyArrayCursor) Close()                       {}
func (c *unsignedEmptyArrayCursor) Stats() cursors.CursorStats   { return cursors.CursorStats{} }
func (c *unsignedEmptyArrayCursor) Next() *cursors.UnsignedArray { return &c.res }

// ********************
// String Array Cursor

type stringArrayFilterCursor struct {
	cursors.StringArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.StringArray
	tmp  *cursors.StringArray
}

func newStringFilterArrayCursor(cond expression) *stringArrayFilterCursor {
	return &stringArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewStringArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.StringArray{},
	}
}

func (c *stringArrayFilterCursor) reset(cur cursors.StringArrayCursor) {
	c.StringArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *stringArrayFilterCursor) Stats() cursors.CursorStats { return c.StringArrayCursor.Stats() }

func (c *stringArrayFilterCursor) Next() *cursors.StringArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.StringArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.StringArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}
		// Clear buffered timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil
		a = c.StringArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type stringMultiShardArrayCursor struct {
	cursors.StringArrayCursor
	cursorContext
	filter *stringArrayFilterCursor
}

func (c *stringMultiShardArrayCursor) reset(cur cursors.StringArrayCursor, itrs cursors.CursorIterators, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newStringFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.StringArrayCursor = cur
	c.itrs = itrs
	c.err = nil
	c.count = 0
}

func (c *stringMultiShardArrayCursor) Err() error { return c.err }

func (c *stringMultiShardArrayCursor) Stats() cursors.CursorStats {
	return c.StringArrayCursor.Stats()
}

func (c *stringMultiShardArrayCursor) Next() *cursors.StringArray {
	for {
		a := c.StringArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		c.count += int64(a.Len())
		if c.count > c.limit {
			diff := c.count - c.limit
			c.count -= diff
			rem := int64(a.Len()) - diff
			a.Timestamps = a.Timestamps[:rem]
			a.Values = a.Values[:rem]
		}
		return a
	}
}

func (c *stringMultiShardArrayCursor) nextArrayCursor() bool {
	if len(c.itrs) == 0 {
		return false
	}

	c.StringArrayCursor.Close()

	var itr cursors.CursorIterator
	var cur cursors.Cursor
	for cur == nil && len(c.itrs) > 0 {
		itr, c.itrs = c.itrs[0], c.itrs[1:]
		cur, _ = itr.Next(c.ctx, c.req)
	}

	var ok bool
	if cur != nil {
		var next cursors.StringArrayCursor
		next, ok = cur.(cursors.StringArrayCursor)
		if !ok {
			cur.Close()
			next = StringEmptyArrayCursor
			c.itrs = nil
			c.err = errors.New("expected string cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.StringArrayCursor = next
	} else {
		c.StringArrayCursor = StringEmptyArrayCursor
	}

	return ok
}

type integerStringCountArrayCursor struct {
	cursors.StringArrayCursor
}

func (c *integerStringCountArrayCursor) Stats() cursors.CursorStats {
	return c.StringArrayCursor.Stats()
}

func (c *integerStringCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.StringArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.StringArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type stringEmptyArrayCursor struct {
	res cursors.StringArray
}

var StringEmptyArrayCursor cursors.StringArrayCursor = &stringEmptyArrayCursor{}

func (c *stringEmptyArrayCursor) Err() error                 { return nil }
func (c *stringEmptyArrayCursor) Close()                     {}
func (c *stringEmptyArrayCursor) Stats() cursors.CursorStats { return cursors.CursorStats{} }
func (c *stringEmptyArrayCursor) Next() *cursors.StringArray { return &c.res }

// ********************
// Boolean Array Cursor

type booleanArrayFilterCursor struct {
	cursors.BooleanArrayCursor
	cond expression
	m    *singleValue
	res  *cursors.BooleanArray
	tmp  *cursors.BooleanArray
}

func newBooleanFilterArrayCursor(cond expression) *booleanArrayFilterCursor {
	return &booleanArrayFilterCursor{
		cond: cond,
		m:    &singleValue{},
		res:  cursors.NewBooleanArrayLen(MaxPointsPerBlock),
		tmp:  &cursors.BooleanArray{},
	}
}

func (c *booleanArrayFilterCursor) reset(cur cursors.BooleanArrayCursor) {
	c.BooleanArrayCursor = cur
	c.tmp.Timestamps, c.tmp.Values = nil, nil
}

func (c *booleanArrayFilterCursor) Stats() cursors.CursorStats { return c.BooleanArrayCursor.Stats() }

func (c *booleanArrayFilterCursor) Next() *cursors.BooleanArray {
	pos := 0
	c.res.Timestamps = c.res.Timestamps[:cap(c.res.Timestamps)]
	c.res.Values = c.res.Values[:cap(c.res.Values)]

	var a *cursors.BooleanArray

	if c.tmp.Len() > 0 {
		a = c.tmp
	} else {
		a = c.BooleanArrayCursor.Next()
	}

LOOP:
	for len(a.Timestamps) > 0 {
		for i, v := range a.Values {
			c.m.v = v
			if c.cond.EvalBool(c.m) {
				c.res.Timestamps[pos] = a.Timestamps[i]
				c.res.Values[pos] = v
				pos++
				if pos >= MaxPointsPerBlock {
					c.tmp.Timestamps = a.Timestamps[i+1:]
					c.tmp.Values = a.Values[i+1:]
					break LOOP
				}
			}
		}
		// Clear buffered timestamps & values if we make it through a cursor.
		// The break above will skip this if a cursor is partially read.
		c.tmp.Timestamps = nil
		c.tmp.Values = nil
		a = c.BooleanArrayCursor.Next()
	}

	c.res.Timestamps = c.res.Timestamps[:pos]
	c.res.Values = c.res.Values[:pos]

	return c.res
}

type booleanMultiShardArrayCursor struct {
	cursors.BooleanArrayCursor
	cursorContext
	filter *booleanArrayFilterCursor
}

func (c *booleanMultiShardArrayCursor) reset(cur cursors.BooleanArrayCursor, itrs cursors.CursorIterators, cond expression) {
	if cond != nil {
		if c.filter == nil {
			c.filter = newBooleanFilterArrayCursor(cond)
		}
		c.filter.reset(cur)
		cur = c.filter
	}

	c.BooleanArrayCursor = cur
	c.itrs = itrs
	c.err = nil
	c.count = 0
}

func (c *booleanMultiShardArrayCursor) Err() error { return c.err }

func (c *booleanMultiShardArrayCursor) Stats() cursors.CursorStats {
	return c.BooleanArrayCursor.Stats()
}

func (c *booleanMultiShardArrayCursor) Next() *cursors.BooleanArray {
	for {
		a := c.BooleanArrayCursor.Next()
		if a.Len() == 0 {
			if c.nextArrayCursor() {
				continue
			}
		}
		c.count += int64(a.Len())
		if c.count > c.limit {
			diff := c.count - c.limit
			c.count -= diff
			rem := int64(a.Len()) - diff
			a.Timestamps = a.Timestamps[:rem]
			a.Values = a.Values[:rem]
		}
		return a
	}
}

func (c *booleanMultiShardArrayCursor) nextArrayCursor() bool {
	if len(c.itrs) == 0 {
		return false
	}

	c.BooleanArrayCursor.Close()

	var itr cursors.CursorIterator
	var cur cursors.Cursor
	for cur == nil && len(c.itrs) > 0 {
		itr, c.itrs = c.itrs[0], c.itrs[1:]
		cur, _ = itr.Next(c.ctx, c.req)
	}

	var ok bool
	if cur != nil {
		var next cursors.BooleanArrayCursor
		next, ok = cur.(cursors.BooleanArrayCursor)
		if !ok {
			cur.Close()
			next = BooleanEmptyArrayCursor
			c.itrs = nil
			c.err = errors.New("expected boolean cursor")
		} else {
			if c.filter != nil {
				c.filter.reset(next)
				next = c.filter
			}
		}
		c.BooleanArrayCursor = next
	} else {
		c.BooleanArrayCursor = BooleanEmptyArrayCursor
	}

	return ok
}

type integerBooleanCountArrayCursor struct {
	cursors.BooleanArrayCursor
}

func (c *integerBooleanCountArrayCursor) Stats() cursors.CursorStats {
	return c.BooleanArrayCursor.Stats()
}

func (c *integerBooleanCountArrayCursor) Next() *cursors.IntegerArray {
	a := c.BooleanArrayCursor.Next()
	if len(a.Timestamps) == 0 {
		return &cursors.IntegerArray{}
	}

	ts := a.Timestamps[0]
	var acc int64
	for {
		acc += int64(len(a.Timestamps))
		a = c.BooleanArrayCursor.Next()
		if len(a.Timestamps) == 0 {
			res := cursors.NewIntegerArrayLen(1)
			res.Timestamps[0] = ts
			res.Values[0] = acc
			return res
		}
	}
}

type booleanEmptyArrayCursor struct {
	res cursors.BooleanArray
}

var BooleanEmptyArrayCursor cursors.BooleanArrayCursor = &booleanEmptyArrayCursor{}

func (c *booleanEmptyArrayCursor) Err() error                  { return nil }
func (c *booleanEmptyArrayCursor) Close()                      {}
func (c *booleanEmptyArrayCursor) Stats() cursors.CursorStats  { return cursors.CursorStats{} }
func (c *booleanEmptyArrayCursor) Next() *cursors.BooleanArray { return &c.res }
