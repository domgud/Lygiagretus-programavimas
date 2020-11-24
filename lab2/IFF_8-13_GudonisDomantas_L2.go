package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

var (
	DataSize = 1000
	WorkerCount = DataSize / 4
)

type Students struct {
	Students []Student `json:"students"`
}
type Student struct {
	Name  string    `json:"name"`
	LastName string `json:lastname`
	Year  int       `json:"year"`
	Grade float32   `json:"grade"`

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
	ResultValue [32]byte // fancy byte hash for now
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

func Method(dataToWorkerChan <-chan *Student, workerToResultChan chan<- *Result, workerRequestChan, workerExitChan chan<- int){

	for {
		workerRequestChan<- 1
		stud := <-dataToWorkerChan
		if stud == nil {
			// No more work to be done, so its time to exit :^)
			break
		}

		stringToHash := fmt.Sprintf("%v %v %v %v", stud.Name, stud.LastName, stud.Year, stud.Grade)
		var hash [32]byte
		hash = sha256.Sum256([]byte(stringToHash)) // Fancy hashing
		for i := 0; i < 1000; i++ {
			hash = sha256.Sum256([]byte(fmt.Sprintf("%v%x", i, hash)))
		}

		if stud.Grade <= 5 { // Very fancy filtering
			workerToResultChan<- &Result{stud,hash}
		}
	}
	workerExitChan<- 0 // Sends a signal to the main thread
}

func DataWorker(mainToDataChan <-chan Student, dataToWorkerChan chan<- *Student, workerRequestChan, mainRequestChan chan int){
	arraySize := DataSize / 2
	localArray := make([]Student, arraySize)
	count := 0
	isDone := false

	for !isDone || count > 0 {

		if count >= arraySize { // If array is at maximum capacity, wait for the worker thread to take data out
			<-workerRequestChan
			value := localArray[count - 1]
			dataToWorkerChan <- &value
			count--
		} else if count <= 0 && !isDone{ // If array count is 0 or less, wait for the main thread to add a value
			<-mainRequestChan
			localArray[count] = <-mainToDataChan
			count++
		} else {
			select { // Regular working conditions
			case request := <-mainRequestChan: // Adding request
				if request == 0 {
					isDone = true
					break
				}
				input := <-mainToDataChan
				localArray[count] = input
				count++
			case <-workerRequestChan: // Removing request
				value := localArray[count - 1]
				dataToWorkerChan <- &value
				count--
				break
			}
		}
	}

	close(dataToWorkerChan) // Closing the data -> worker channel (worker will start receiving nil)
	for i := 0 ; i < WorkerCount; i++ {
		<-workerRequestChan // Consuming the worker requests so they can take nil's from workerChan and exit
	}
	close(workerRequestChan)
	fmt.Println("Data thread finished")
}

func ResultWorker(workerToResultChan <-chan *Result, resultToMainChan chan<- []Result, resultCountChan chan<- int) {
	localArray := make([]Result, DataSize)
	counter := 0

	for {
		result := <-workerToResultChan
		if result == nil {
			break
		}
		addedInLoop := false

		for index := 0; index < counter; index++ {
			if localArray[index].Student.Compare(result.Student) {
				var oldRez Result
				newRez := *result
				for i := index; i < counter + 1; i++ {
					oldRez = localArray[i]
					localArray[i] = newRez
					newRez = oldRez
				}
				counter++
				addedInLoop = true
				break
			}
		}
		if !addedInLoop {
			localArray[counter] = *result
			counter++
		}
	}
	fmt.Println("Exiting result thread and sending data back")
	resultToMainChan<- localArray
	resultCountChan<- counter
}

func main(){

	t := time.Now()
	var students Students
	students.ReadJsonStudents("IFF-8-13_GudonisD_L1a_dat2.json")

	mainRequestChan := make(chan int) // Request channels
	workerRequestChan := make(chan int)

	mainToDataChan := make(chan Student) // Data transfer channels
	dataToWorkerChan := make(chan *Student)
	workerToResultChan := make(chan *Result)
	workerExitChan := make(chan int)
	resultToMainChan := make(chan []Result)
	resultCountChan := make(chan int)

	for i := 0; i < WorkerCount; i++ {
		go Method(dataToWorkerChan, workerToResultChan, workerRequestChan, workerExitChan)
	}

	go DataWorker(mainToDataChan, dataToWorkerChan, workerRequestChan, mainRequestChan)
	go ResultWorker(workerToResultChan, resultToMainChan, resultCountChan)

	for _, stud := range students.Students { // Adding values to the DataMonitor
		mainRequestChan<- 1
		mainToDataChan<- stud
	}
	close(mainRequestChan)

	for i := 0 ; i < WorkerCount; i++ {
		<-workerExitChan // Waiting for all workers to finish
	}
	fmt.Println("All workers exited")
	close(workerToResultChan)



	// Creating and writing the result file ------------------------------------------------------------

	// Taking from the result thread after the workers have finished
	results := <-resultToMainChan
	count := <-resultCountChan

	resultFile, err := os.OpenFile("IFF-8-13_GudonisD_L1a_rez2.txt",os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer resultFile.Close()

	resultFile.WriteString(fmt.Sprintf("Name: %-12v |UserName: %-10v |Year: %-5v |Grade: %-5v |Hash: %v\n","", "","","",""))

	for i := 0; i < count; i++{
		resultFile.WriteString(fmt.Sprintf("%-18v |%-20v |%-11v |%-12v |%x\n",
			results[i].Student.Name,
			results[i].Student.LastName,
			results[i].Student.Year,
			results[i].Student.Grade,
			results[i].ResultValue))
	}
	fmt.Println("Total time taken:",time.Now().Sub(t))
	fmt.Println("I die peacefully")
}



