package main
import "fmt"

func main(){
	// var name string = "Shwejit Vasu"
	// age:= 23
	
	// fmt.Println(age)
	// fmt.Println(name)

	// var age int
	// var name string
	// fmt.Println("Enter the age: ")
	// fmt.Scanln(&age)

	// ages:=[]int{15,16,17,18,19,20}

	// for _, age:= range ages{
	// 	fmt.Println("Age", age)
	// 	var name string
	// 	fmt.Print("Enter the name: ")
	// 	fmt.Scanln(&name)
	// 	fmt.Println(canDrive(age,name))
	// }

	// for i:=0;i<len(ages);i++{
	// 	fmt.Println("Age", ages[i])
	// 	var name string
	// 	fmt.Println("Enter the name: ")
	// 	fmt.Scanln(&name)
	// 	fmt.Println(canDrive(ages[i],name))
	// }

	// append
	
	// arr:= append(age, 21)
	// fmt.Println(arr)
	// fmt.Println(age)
	// fmt.Println(len(age))

	// Loop

	// for i:=15;i<=20;i++{
	// 	var name string
	// 	fmt.Println("Enter the name: ")
	// 	fmt.Scanln(&name)
	// 	fmt.Println(canDrive(i,name))
	// }

	// msg:= canDrive(age, name)
	// fmt.Println(msg)


	// Enter the age using command line
	var n int
	fmt.Print("No of persons: ")
	fmt.Scanln(&n)
	ages:=make([]int, n)

	for _, age:=range ages{
		fmt.Print("Enter the age of the Person: ")
		fmt.Scanln(&age)
		var name string
		fmt.Print("Enter the name of the Person ")
		fmt.Scanln(&name)

		fmt.Println(canDrive(age, name))
	}

}

func canDrive(age int, name string) string{

	// var msg string
	if age >= 18{
		return name + " You are eligible to Drive"
		// fmt.Println(name,"You are eligible to Drive")
	} else {
		return name + " You are not eligible to Drive"
		// fmt.Println(name,"You are not eligible to Drive")
	}
}
