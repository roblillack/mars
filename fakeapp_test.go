package mars

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
)

type Hotel struct {
	HotelID          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

type Hotels struct {
	*Controller
}

type MyStatic struct {
	*Controller
}

func (c Hotels) Show(id int) Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.Render(Args{"title": title, "hotel": hotel})
}

func (c Hotels) Book(id int) Result {
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.RenderJSON(hotel)
}

func (c Hotels) Index() Result {
	return c.RenderText("Hello, World!")
}

func (c MyStatic) Serve(prefix, filepath string) Result {
	var basePath, dirName string

	if !path.IsAbs(dirName) {
		basePath = BasePath
	}

	fname := path.Join(basePath, prefix, filepath)
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return c.NotFound("")
	} else if err != nil {
		WARN.Printf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}
	return c.RenderFile(file, "")
}

func startFakeBookingApp() {
	RegisterController((*Hotels)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Index",
			},
			&MethodType{
				Name: "Show",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
			},
			&MethodType{
				Name: "Book",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
			},
		})

	RegisterController((*Static)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Serve",
				Args: []*MethodArg{
					&MethodArg{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					&MethodArg{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
			},
		})

	// Disable logging.
	_ = ioutil.Discard
	TRACE = log.New(os.Stderr, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE

	InitDefaults("prod", "testdata")

	runStartupHooks()
}
