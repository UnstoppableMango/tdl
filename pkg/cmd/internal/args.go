package internal

func SplitAt(args []string, dash int) (l []string, r []string) {
	if dash == -1 {
		return args, []string{}
	}

	for i, a := range args {
		if i < dash {
			l = append(l, a)
		} else {
			r = append(r, a)
		}
	}

	return
}
