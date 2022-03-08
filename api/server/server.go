package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"todo/todopb"

	objectid "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)
 
func checkErr(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}
 
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
 
type server struct{}
 
// Mongoのコレクションを設定
var collection *mongo.Collection
 
// Toodのアイテムを設定
type todoItem struct {
	ID       objectid.ObjectID `bson: "_id,omitempty`
	AuthorID string            `bson:"author_id"`
	Content  string            `bson:"content"`
	Title    string            `bson:"title"`
}
 
func main() {
	fmt.Println("=======Todo API Start====")
 
	// Connecting Mongo
	// mongoは、docker-compose.ymlのservice名
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://mongo:27017"))
	checkErr("Faile to create mongo client: %v", err)
	err = client.Connect(context.Background())
	checkErr("Cannot connect to mongo db: %v", err)
	collection = client.Database("mydb").Collection("blog")
 
	// 50051ポートで起動する(docker-compose.ymlと合わせる)
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	checkErr("Failed to listen: %v", err)
 
	// grpcのサーバーを作成
	s := grpc.NewServer()
	// TodoServiceServerにマウント
	todopb.RegisterTodoServiceServer(s, &server{})
 
	go func() {
		fmt.Println("Starting Server...")
		err = s.Serve(lis)
		checkErr("faild to server: %v", err)
	}()
 
	// Ctrl + cでプログラムから抜けられるようにする
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
 
	// シグナルが受け取れるまで、ブロック
	<-ch
 
	fmt.Println("Stopping the server")
	s.Stop()
	fmt.Println("Closing the listener")
	lis.Close()
	fmt.Println("=====Todo API End====")
}

func dataToTodoPb(data *todoItem) *todopb.Todo {
	return &todopb.Todo{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Title:    data.Title,
		Content:  data.Content,
	}
}

func (*server) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
	fmt.Printf("Create Todo Item with %v\n", req)
 
	// requestにあるTodoデータを取得
	todo := req.GetTodo()
	data := todoItem{
		AuthorID: todo.GetAuthorId(),
		Title:    todo.GetTitle(),
		Content:  todo.GetContent(),
	}
	// MongoDBへデータを挿入
	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
 
	// Mongoで自動生成されるidを取得
	oid, ok := res.InsertedID.(objectid.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}
	data.ID = oid
 
	return &todopb.CreateTodoResponse{
		Todo: dataToTodoPb(&data),
	}, nil
 
}

func (*server) ReadTodo(ctx context.Context, req *todopb.ReadTodoRequest) (*todopb.ReadTodoResponse, error) {
	fmt.Printf("Read Todo Item with %v\n", req)
	id := req.GetTodoId()
	oid, err := objectid.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	todo := &todoItem{}
	filter := objectid.D{{"_id", oid}}
	if err := collection.FindOne(context.Background(), filter).Decode(todo); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with specified ID: %v\n", err),
		)
	}
        todo.ID = oid
	return &todopb.ReadTodoResponse{
		Todo: dataToTodoPb(todo),
	}, nil
}

func (*server) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.UpdateTodoResponse, error) {
	fmt.Printf("Update Todo Item with %v\n", req)
	todo := req.GetTodo()
	oid, err := objectid.ObjectIDFromHex(todo.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	data := &todoItem{}
	filter := objectid.D{{"_id", oid}}

	if err := collection.FindOne(context.Background(), filter).Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with specified ID: %v\n", err),
		)
	}

	data.ID = oid
	data.AuthorID = todo.GetAuthorId()
	data.Title = todo.GetTitle()
	data.Content = todo.GetContent()

	_, updateErr := collection.ReplaceOne(context.Background(), filter, data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
		)
	}

	return &todopb.UpdateTodoResponse{
		Todo: dataToTodoPb(data),
	}, nil
}


func (*server) DeleteTodo(ctx context.Context, req *todopb.DeleteTodoRequest) (*todopb.DeleteTodoResponse, error) {
	fmt.Printf("Delete todo request with %v\n", req)

	oid, err := objectid.ObjectIDFromHex(req.GetTodoId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	filter := objectid.D{{"_id", oid}}

	res, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete object in MongoDB: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo in MongoDB: %v", err),
		)
	}

	return &todopb.DeleteTodoResponse{TodoId: req.GetTodoId()}, nil
}

func (*server) ListTodo(req *todopb.ListTodoRequest, stream todopb.TodoService_ListTodoServer) error {
	fmt.Printf("List todo request with %v\n", req)
 
	cur, err := collection.Find(context.Background(), objectid.D{})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unkown internal error: %v", err),
		)
	}
	defer cur.Close(context.Background())
 
	for cur.Next(context.Background()) {
		data := &todoItem{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)
		}
		// Straming 配信
		stream.Send(&todopb.ListTodoResponse{Todo: dataToTodoPb(data)})
	}
 
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unkown internal error: %v", err),
		)
	}
 
	return nil
}
