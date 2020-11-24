package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"hash/adler32"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

var (
	DataSize = 1000
)

type Students struct {
	Students []Student `json:"students"`
}
type Student struct {
	Name     string  `json:"name"`
	LastName string  `json:"lastname"`
	Year     int     `json:"year"`
	Grade    float32 `json:"grade"`

}
func (student *Student) Compare(other *Student) bool{
	if student.Year == other.Year {
		return student.Grade >  other.Grade
	} else {
		return student.Year > other.Year
	}
}
type Result struct {
	Student *Student
	ResultValue uint32
}

type DataMonitor struct {
	DataArray         []Student
	Count             int
	hasFinishedAdding bool
	cond              *sync.Cond
	mutex             *sync.Mutex
}
type ResultMonitor struct {
	DataArray         []Result
	Count             int
	cond              *sync.Cond
	mutex             *sync.Mutex
}



func CreateDataMonitor() *DataMonitor {
	mutex := sync.Mutex{}
	return &DataMonitor{Count: 0, cond: sync.NewCond(&mutex), mutex: &mutex, DataArray: make([]Student, DataSize / 2)}
}

func CreateResultMonitor() *ResultMonitor {
	mutex := sync.Mutex{}
	return &ResultMonitor{Count: 0, cond: sync.NewCond(&mutex), mutex: &mutex, DataArray: make([]Result, DataSize)}
}

func (masyvas *ResultMonitor) Add(result *Result) {

	masyvas.mutex.Lock()
	defer masyvas.mutex.Unlock()

	for index := 0; index < masyvas.Count; index++ {
		if masyvas.DataArray[index].Student.Compare(result.Student) {

			var oldRez Result
			newRez := *result
			for i := index; i < masyvas.Count + 1; i++ {
				oldRez = masyvas.DataArray[i]
				masyvas.DataArray[i] = newRez
				newRez = oldRez
			}
			masyvas.Count++
			return
		}
	}
	masyvas.DataArray[masyvas.Count] = *result
	masyvas.Count++
}

func (masyvas *DataMonitor) Add(student *Student){

	masyvas.mutex.Lock()
	defer masyvas.mutex.Unlock() // atrakina kritine sekcija pasibaigus funkcijai
	defer masyvas.cond.Signal()  // pabutina threada kad pasiimtu reiksme

	for masyvas.Count >= len(masyvas.DataArray) {
		// uzmigdomas thread laukia kol bus isimta reiksme
		masyvas.cond.Wait()

	}
	masyvas.DataArray[masyvas.Count] = *student
	masyvas.Count++

}

func (masyvas *DataMonitor) Take() *Student{

	masyvas.mutex.Lock() // uzrakina kritine sekcija

	defer masyvas.mutex.Unlock() // atrakina kritine sekcija pabaigoje
	defer masyvas.cond.Signal() //pranesa, kad reiksme paimta

	for masyvas.Count == 0 {
		if masyvas.hasFinishedAdding {
			// grazinama null, kad zinotu jog daugiau nera dedamos reiksmes i monitoriu
			return nil
		}
		masyvas.cond.Wait()
	}

	stud := masyvas.DataArray[masyvas.Count - 1]
	masyvas.Count--

	return &stud
}

func (students *Students) ReadJsonStudents(fileName string){
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	err = json.Unmarshal(byteValue, &students)
	if err != nil {
		panic(err)
	}
}

func HashingMethod(data *DataMonitor, result *ResultMonitor,wg *sync.WaitGroup){

	defer wg.Done()

	for { // Loopinama per data monitor ir imama reiksmes
		stud := data.Take()
		if stud == nil {
			// Daugiau nera reiksmu
			break
		}
		stringToHash := fmt.Sprintf("%v %v %v %v", stud.Name, stud.Year, stud.Grade, stud.LastName)
		var hash []byte
		var hash2 uint32
		var a = fnv.New32()
		hash = a.Sum([]byte(stringToHash))
		for i := 0; i < 3000; i++ {
			hash2 = adler32.Checksum([]byte(fmt.Sprintf("%v%x", i, hash)))
		}

		if stud.Grade <= 5 {
			result.Add(&Result{Student: stud, ResultValue: hash2})
		}

	}
}

func PrintToFile(resultMonitor *ResultMonitor){
	resultFile, err := os.OpenFile("rez.txt",os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer resultFile.Close()

	resultFile.WriteString(fmt.Sprintf("Name: %-12v |LastName: %-10v |Year: %-5v |Grade: %-5v |Hash: %v\n","", "","","",""))
	for i := 0; i < resultMonitor.Count; i++{
		resultFile.WriteString(fmt.Sprintf("%-18v |%-20v |%-11v |%-12v |%x\n",
			resultMonitor.DataArray[i].Student.Name,
			resultMonitor.DataArray[i].Student.LastName,
			resultMonitor.DataArray[i].Student.Year,
			resultMonitor.DataArray[i].Student.Grade,
			resultMonitor.DataArray[i].ResultValue))
	}

}

func main(){

	t := time.Now()
	var students Students
	dataMonitor := CreateDataMonitor()
	resultMonitor := CreateResultMonitor()
	students.ReadJsonStudents("1.json")
	wGroup := sync.WaitGroup{}
	threadCount := DataSize/4
	wGroup.Add(threadCount)

	for i := 0; i < threadCount; i++ {
		go HashingMethod(dataMonitor, resultMonitor,&wGroup)
	}

	for _, stud := range students.Students { // sudedamos vertes i monitoriu
		dataMonitor.Add(&stud)
	}
	dataMonitor.hasFinishedAdding = true // nustatoma, kad daugiau nebus dedama verciu

	wGroup.Wait()

	PrintToFile(resultMonitor)

	fmt.Println("Total time taken:",time.Now().Sub(t))
	fmt.Println("Done")
}



