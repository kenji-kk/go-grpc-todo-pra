package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"todo/todopb"

	"google.golang.org/grpc"
)
 
func checkError(message string, err error) {
	if err != nil {
		log.Fatalf("message", err)
	}
}
 
func createTodoHandler(c todopb.TodoServiceClient) string {
	fmt.Println("Creating the todo")
 
	// Todoデータを作成
	todo := &todopb.Todo{
		AuthorId: "selfnote",
		Title:    "First Post",
		Content:  "Fist Post for US!",
	}
 
	createTodoRes, err := c.CreateTodo(context.Background(), &todopb.CreateTodoRequest{Todo: todo})
	checkError("Failt to create todo data: %v\n", err)
	fmt.Printf("Todo has been created: %v\n", createTodoRes)
	return createTodoRes.GetTodo().GetId()
}
 
func readTodoHandler(c todopb.TodoServiceClient, id string) {
	fmt.Println("Read the todo with id")
 
	readTodoRes, err := c.ReadTodo(context.Background(), &todopb.ReadTodoRequest{TodoId: id})
	checkError("Error happend while reading: %v\n", err)
	fmt.Printf("Todo was read: %v\n", readTodoRes)
}


func updateTodoHandler(c todopb.TodoServiceClient, id string) {
	fmt.Println("Update the todo with id")
	newTodo := &todopb.Todo{
		Id:       id,
		AuthorId: "Change Author",
		Title:    "First Post(edit)",
		Content:  "First Post for US!(edit)",
	}
	updateRes, updateErr := c.UpdateTodo(context.Background(), &todopb.UpdateTodoRequest{Todo: newTodo})
	checkError("Error happend while updating: %v\n", updateErr)

	fmt.Printf("Blog was updated: %v\n", updateRes)
}


func deleteTodoHandler(c todopb.TodoServiceClient, id string) {
	fmt.Println("Delete the todo with id")
	deleteRes, deleteErr := c.DeleteTodo(
		context.Background(),
		&todopb.DeleteTodoRequest{TodoId: id})
	checkError("Error happend while deleting: %v\n", deleteErr)
	fmt.Printf("Todo was deleted: %v\n", deleteRes)
}

func listTodoHandler(c todopb.TodoServiceClient) {
	fmt.Println("List the todo")
	stream, err := c.ListTodo(context.Background(), &todopb.ListTodoRequest{})
	checkError("error while calling ListTodo RPC: %v\n", err)
 
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		checkError("Something happend: %v\n", err)
		fmt.Println(res.GetTodo())
	}
}
 
func main() {
	opts := grpc.WithInsecure()
 
	// serverは、docker-compose.ymlに定義してあるサービス名
	cc, err := grpc.Dial("server:50051", opts)
	checkError("could not connect: %v\n", err)
	defer cc.Close()
	c := todopb.NewTodoServiceClient(cc)
 
	// Create Todo
	todoId := createTodoHandler(c)
	readTodoHandler(c, todoId)
	updateTodoHandler(c, todoId)
	listTodoHandler(c)
	deleteTodoHandler(c, todoId)
}
