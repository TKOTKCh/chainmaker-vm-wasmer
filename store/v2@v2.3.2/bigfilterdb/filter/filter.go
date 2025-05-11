package filter

// Filter support add key , check key is/not exist
type Filter interface {

	// Add
	//  @Description: a key to filter
	//  @param key
	//  @return error
	Add(key []byte) error

	// ExistMult
	//  @Description: Check more keys is/not exist
	//  @param arr
	//  @return []bool
	ExistMult(arr [][]byte) []bool

	// Exist
	//  @Description: Check key is/not exist
	//  @param key
	//  @return bool
	//  @return error
	Exist(key []byte) (bool, error)

	// AddMult
	//  @Description: Add more key to filter
	//  @param arr
	//  @return error
	AddMult(arr [][]byte) error

	// SaveLastPoint
	//  @Description: Save last point to server
	//  @param pointK
	//  @param pointV
	//  @return error
	SaveLastPoint(pointK string, pointV string) error

	// GetLastPoint
	//  @Description: get last point to server
	//  @param pointK
	//  @return []string
	//  @return error
	GetLastPoint(pointK string) ([]string, error)

	//Dump(b []byte) error
	//
	//Recover()
}
