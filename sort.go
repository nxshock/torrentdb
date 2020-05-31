package main

type SortField string

const (
	FieldName SortField = "name"
	FieldSize SortField = "size"
	FieldTime SortField = "time"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)
