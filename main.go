package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

// Product structure represents a product in the store
type Product struct {
	ID    int
	Name  string
	Size  string
	Price float64
}

var db *sql.DB

func initDB() *sql.DB {
	// Replace with your actual PostgreSQL connection details
	connStr := "user=postgres password=rayana2015 dbname=newdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error opening database connection:", err)
		panic(err)
	}

	// Ensure the database connection is successful
	err = db.Ping()
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		panic(err)
	}

	fmt.Println("Connected to the database")

	return db
}

func fetchProductsFromDB(filter string) ([]Product, error) {
	var products []Product

	var query string
	var args []interface{}

	if filter != "" {
		query = "SELECT id, name, size, price FROM products WHERE name ILIKE $1"
		args = append(args, "%"+filter+"%")
	} else {
		query = "SELECT id, name, size, price FROM products"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println("Error fetching products from the database:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Size, &p.Price); err != nil {
			fmt.Println("Error scanning product row:", err)
			continue
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error iterating over product rows:", err)
		return nil, err
	}

	return products, nil
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")

	products, err := fetchProductsFromDB(filter)
	if err != nil {
		http.Error(w, "Error fetching products from the database", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("index").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Online Store</title>
		</head>
		<body>
			<h1>Welcome to the Online Store!</h1>
			<form action="/" method="get">
				<label for="filter">Filter:</label>
				<input type="text" id="filter" name="filter" placeholder="Enter filter" value="{{.Filter}}">
				<button type="submit">Apply Filter</button>
			</form>
			<h2>Products:</h2>
			<table border="1">
				<tr>
					<th>ID</th>
					<th>Name</th>
					<th>Size</th>
					<th>Price</th>
					<th>Action</th>
				</tr>
				{{range .Products}}
					<tr>
						<td>{{.ID}}</td>
						<td>{{.Name}}</td>
						<td>{{.Size}}</td>
						<td>${{.Price}}</td>
						<td>
							<form method="post" action="/delete/{{.ID}}">
								<input type="hidden" name="_method" value="DELETE">
								<button type="submit">Delete</button>
							</form>
							<form method="get" action="/edit/{{.ID}}">
								<button type="submit">Edit</button>
							</form>
						</td>
					</tr>
				{{end}}
			</table>
			<a href="/add-product">Add Product</a>
		</body>
		</html>
	`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Filter   string
		Products []Product
	}{
		Filter:   filter,
		Products: products,
	}

	tmpl.Execute(w, data)
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/delete/"):]
	productID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM products WHERE id = $1", productID)
	if err != nil {
		fmt.Println("Error deleting from database:", err)
		http.Error(w, "Error deleting from database", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Product deleted with ID: %d\n", productID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func AddProductHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("addProduct").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Add Product</title>
		</head>
		<body>
			<h1>Add a new product</h1>
			<form method="post" action="/add-product-post">
				<label for="name">Name:</label>
				<input type="text" name="name" required><br>
				<label for="size">Size:</label>
				<input type="text" name="size" required><br>
				<label for="price">Price:</label>
				<input type="number" name="price" step="0.01" required><br>
				<input type="submit" value="Add Product">
			</form>
			<a href="/">Back to Home</a>
		</body>
		</html>
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

func AddProductPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	_, err := db.Exec("INSERT INTO products (name, size, price) VALUES ($1, $2, $3)",
		r.FormValue("name"), r.FormValue("size"), r.FormValue("price"))
	if err != nil {
		fmt.Println("Error inserting into database:", err)
		http.Error(w, "Error inserting into database", http.StatusInternalServerError)
		return
	}

	fmt.Printf("New product added: Name=%s, Size=%s, Price=%s\n", r.FormValue("name"), r.FormValue("size"), r.FormValue("price"))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func EditProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/edit/"):]
	productID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Fetch the product details from the database
	var product Product
	err = db.QueryRow("SELECT id, name, size, price FROM products WHERE id = $1", productID).
		Scan(&product.ID, &product.Name, &product.Size, &product.Price)
	if err != nil {
		fmt.Println("Error fetching product details:", err)
		http.Error(w, "Error fetching product details", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("editProduct").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>Edit Product</title>
        </head>
        <body>
            <h1>Edit Product</h1>
            <form method="post" action="/edit-product-post/{{.ID}}">
                <label for="name">Name:</label>
                <input type="text" name="name" value="{{.Name}}" required><br>
                <label for="size">Size:</label>
                <input type="text" name="size" value="{{.Size}}" required><br>
                <label for="price">Price:</label>
                <input type="number" name="price" step="0.01" value="{{.Price}}" required><br>
                <input type="submit" value="Save Changes">
            </form>
            <a href="/">Back to Home</a>
        </body>
        </html>
    `)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, product)
}

func EditProductPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/edit-product-post/"):]
	productID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE products SET name=$1, size=$2, price=$3 WHERE id=$4",
		r.FormValue("name"), r.FormValue("size"), r.FormValue("price"), productID)
	if err != nil {
		fmt.Println("Error updating product in database:", err)
		http.Error(w, "Error updating product in database", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Product updated with ID: %d\n", productID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	db = initDB()
	defer db.Close()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/delete/", DeleteHandler)
	http.HandleFunc("/add-product", AddProductHandler)
	http.HandleFunc("/add-product-post", AddProductPostHandler)
	http.HandleFunc("/edit/", EditProductHandler)
	http.HandleFunc("/edit-product-post/", EditProductPostHandler)

	fmt.Println("Server is running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
