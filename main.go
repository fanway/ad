package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
)

var DATABASE_URL = fmt.Sprintf("postgresql://%s:%s@database:5432/avito", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"))

type Ad struct {
	Name   string   `json:"name,omitempty"`
	Desc   string   `json:"desc,omitempty""`
	Images []string `json:"images,omitempty""`
	Price  int      `json:"price,omitempty""`
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), DATABASE_URL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	q := r.URL.Query()
	id, err := strconv.Atoi(q.Get("id"))
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	fields := q["fields"]
	var name, image string
	var price int
	err = conn.QueryRow(context.Background(), "SELECT name, price, images[1] FROM Ad WHERE id=$1", id).Scan(&name, &price, &image)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	ad := Ad{Name: name, Images: []string{image}, Price: price}
	if len(fields) > 0 {
		var desc string
		var s []string
		fieldsSplit := strings.Split(fields[0], ",")
		sort.Strings(fieldsSplit)

		switch len(fieldsSplit) {
		case 2:
			if fieldsSplit[0] == "description" && fieldsSplit[1] == "images" {
				err = conn.QueryRow(context.Background(), "SELECT description, images[2:] FROM Ad WHERE id=$1", id).Scan(&desc, &s)
			}
		case 1:
			if fieldsSplit[0] == "description" {
				err = conn.QueryRow(context.Background(), "SELECT description FROM Ad WHERE id=$1", id).Scan(&desc)
			} else if fieldsSplit[0] == "images" {
				err = conn.QueryRow(context.Background(), "SELECT images[2:] FROM Ad WHERE id=$1", id).Scan(&s)
			}
		}

		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusBadRequest, "Invalid fields")
			return
		}
		ad.Desc = desc
		ad.Images = append(ad.Images, s...)
	}

	respondWithJSON(w, http.StatusOK, ad)
}

func getAllHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), DATABASE_URL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	vals := r.URL.Query()
	offset := 0
	if _, ok := vals["pagination"]; ok {
		if len(vals["pagination"]) > 0 {
			offset, err = strconv.Atoi(vals["pagination"][0])
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid pagination")
				return
			}
		} else {
			respondWithError(w, http.StatusBadRequest, "Invalid pagination")
		}
	}

	price := vals.Get("price")
	date := vals.Get("date")

	var rows pgx.Rows

	fmt.Println(offset)
	fmt.Println(price)

	mask := 0
	if price != "" {
		mask |= 1 << 1
	}
	if date != "" {
		mask |= 1 << 0
	}
	fmt.Println(mask)
	rows, err = conn.Query(context.Background(), `SELECT name, images[1], price 
												  FROM Ad 
												  ORDER BY CASE WHEN ($1=3 AND $2='asc') THEN price END ASC,
												  		   CASE	WHEN ($1=3 AND $2='desc') THEN price END DESC,
												  		   CASE	WHEN ($1=3 AND $3='asc') THEN created_at END ASC,
												  		   CASE	WHEN ($1=3 AND $3='desc') THEN created_at END DESC,
												  		   CASE	WHEN ($1=2 AND $2='asc') THEN price END ASC,
												  		   CASE	WHEN ($1=2 AND $2='desc') THEN price END DESC, 
												  		   CASE	WHEN ($1=1 AND $3='asc') THEN created_at END ASC, 
												  		   CASE	WHEN ($1=1 AND $3='desc') THEN created_at END DESC
												  LIMIT 10
												  OFFSET $4`, mask, price, date, offset)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Error")
		return
	}

	var s []Ad
	for rows.Next() {
		var name, image string
		var price int
		err = rows.Scan(&name, &image, &price)
		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusBadRequest, "Error")
			return
		}
		s = append(s, Ad{Name: name, Images: []string{image}, Price: price})
	}

	respondWithJSON(w, http.StatusOK, struct {
		Data []Ad `json:"data"`
	}{s})
}
func createHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), DATABASE_URL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	decoder := json.NewDecoder(r.Body)
	var ad Ad
	if err := decoder.Decode(&ad); err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()
	if len(ad.Images) > 3 {
		ad.Images = ad.Images[:3]
	}
	var id int
	err = conn.QueryRow(context.Background(), "INSERT INTO Ad (name, description, images, price) VALUES ($1, $2, $3, $4) RETURNING ID", ad.Name, ad.Desc, ad.Images, ad.Price).Scan(&id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Insert error")
		return

	}
	respondWithJSON(w, http.StatusOK, struct {
		Id int `json:"id"`
	}{id})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conn, err := pgx.Connect(context.Background(), DATABASE_URL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	_, err = conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS Ad (id serial PRIMARY KEY, name VARCHAR(200) NOT NULL, description VARCHAR(1000) NOT NULL, images text[] NOT NULL, price int NOT NULL CHECK(price >= 0), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	r := mux.NewRouter()
	r.HandleFunc("/get", getHandler).Methods("GET")
	r.HandleFunc("/getall", getAllHandler).Methods("GET")
	r.HandleFunc("/create", createHandler).Methods("POST")
	fmt.Println("started")
	log.Fatal(http.ListenAndServe(":8080", r))
}
