package main

///import ""
//
func vtrue[T any] (
							is bool, t T, f T,
			) T {
			/*-*/
			if is { return t }
			return f
}

func init() {
	//
}