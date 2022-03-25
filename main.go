package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
)

var tmpDir string
var recursive bool
var include string
var exclude string
var maxWorkers int

type Params struct {
	LOCATION                          string `default:"us-central"`
	DATASET_LOCATION                  string `default:"us"`
	BIGQUERY_PROJECT_ID               string `binding:"required"`
	COLLECTION_PATH                   string
	DATASET_ID                        string
	TABLE_ID                          string
	WILDCARD_IDS                      bool   `default:false`
	TABLE_PARTITIONING                string `default:"NONE"`
	TIME_PARTITIONING_FIELD           string `default:""`
	TIME_PARTITIONING_FIRESTORE_FIELD string `default:""`
	TIME_PARTITIONING_FIELD_TYPE      string `default:"omit"`
	CLUSTERING                        string `default:""`
	BACKUP_COLLECTION                 string `default:""`
	TRANSFORM_FUNCTION                string `default:""`
}

type FirebaseRC struct {
	Projects struct {
		Default string `json:"default"`
	} `json:"projects"`
}

func main() {
	mkDir(tmpDir)
	commonParams := Params{}

	commonParams.getArgs()

	collections := getCollections(commonParams.BIGQUERY_PROJECT_ID, include, exclude, recursive)

	var wg sync.WaitGroup
	sem := make(chan string, maxWorkers)

	wg.Add(len(collections))
	for _, c := range collections {
		sem <- c
		go worker(&commonParams, c, sem, &wg)
	}
	wg.Wait()
}

func worker(p *Params, c string, ch <-chan string, wg *sync.WaitGroup) {
	defer (*wg).Done()
	cp := getCollectionParams(*p, c)
	o, e := mkOutputFiles(cp)
	deployExtension(cp, o, e)
	loadTable(cp, o, e)
	<-ch
}

func getCollectionParams(p Params, c string) *Params {
	splitPath := strings.Split(c, "/")
	p.COLLECTION_PATH = c
	p.TABLE_ID = splitPath[len(splitPath)-1]
	return &p
}

func mkOutputFiles(p *Params) (*os.File, *os.File) {
	o := fmt.Sprintf("%s/%s/stdout.log", tmpDir, (*p).TABLE_ID)
	e := fmt.Sprintf("%s/%s/stderr.log", tmpDir, (*p).TABLE_ID)

	mkDir(path.Dir(o))
	mkDir(path.Dir(e))

	stdoutfile, err := os.OpenFile(o, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	stderrfile, err := os.OpenFile(e, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return stdoutfile, stderrfile
}

func deployExtension(p *Params, o *os.File, e *os.File) {
	fmt.Printf("Deploying Stream Collections to BQ for %s...\n", (*p).COLLECTION_PATH)

	err := createEnvFile(p)
	if err != nil {
		log.Fatal(err)
	}

	o.WriteString(fmt.Sprintf("\n%s\n", time.Now().Local().String()))
	e.WriteString(fmt.Sprintf("\n%s\n", time.Now().Local().String()))

	cmd := exec.Command("firebase",
		fmt.Sprintf("--project=%s", (*p).BIGQUERY_PROJECT_ID),
		"ext:install",
		fmt.Sprintf("--params=%s/%s/.env", tmpDir, (*p).TABLE_ID),
		"firebase/firestore-bigquery-export",
		"--force",
		"--non-interactive")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("firestore-bigquery-export-%s", (*p).TABLE_ID))
	cmd.Stdout = o
	cmd.Stderr = e

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func loadTable(p *Params, o *os.File, e *os.File) {
	fmt.Printf("Loading existing collection data in %s into %s...\n", (*p).COLLECTION_PATH, (*p).TABLE_ID)

	o.WriteString(fmt.Sprintf("\n%s\n", time.Now().Local().String()))
	e.WriteString(fmt.Sprintf("\n%s\n", time.Now().Local().String()))

	cmd := exec.Command("npx",
		"@firebaseextensions/fs-bq-import-collection",
		"--non-interactive",
		fmt.Sprintf("--project=%s", (*p).BIGQUERY_PROJECT_ID),
		fmt.Sprintf("--source-collection-path=%s", (*p).COLLECTION_PATH),
		fmt.Sprintf("--dataset=%s", (*p).DATASET_ID),
		fmt.Sprintf("--table-name-prefix=%s", (*p).TABLE_ID),
		"--multi-threaded",
		fmt.Sprintf("--dataset-location=%s", (*p).DATASET_LOCATION),
		fmt.Sprintf("--query-collection-group=false"),
		"--batch-size=300")

	cmd.Stdout = o
	cmd.Stderr = e

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func createEnvFile(p *Params) error {
	v := reflect.ValueOf(*p)
	typeOfP := v.Type()
	env := make(map[string]string, 0)

	for i := 0; i < v.NumField(); i++ {
		env[fmt.Sprintf("%s", typeOfP.Field(i).Name)] = fmt.Sprintf("%v", v.Field(i).Interface())
	}

	return godotenv.Write(env, fmt.Sprintf("%s/%s/.env", tmpDir, (*p).TABLE_ID))
}

func mkDir(d string) {
	if err := os.MkdirAll(d, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}

func rmTmpDir() {
	if err := os.Remove(tmpDir); err != nil {
		log.Printf("Failed to cleanup tmp files at %s", tmpDir)
	}
}

func getCollections(project string, i string, e string, r bool) []string {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, project)
	if err != nil {
		log.Fatal(err)
	}
	include := strings.Split(i, ",")
	exclude := strings.Split(e, ",")

	var cols []string

	if include[0] == "ALL" {
		iter := client.Collections(ctx)

		for {
			collRef, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			if !contains(exclude, collRef.ID) {
				cols = append(cols, collRef.ID)
			}
		}
	} else {
		cols = include
	}

	return cols
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getFirebaseRC() string {
	f, err := os.Open(".firebaserc")

	if err != nil {
		return ""
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)

	if err != nil {
		return ""
	}

	var rc FirebaseRC

	err = json.Unmarshal(bytes, &rc)

	if err != nil {
		log.Print("Unable to unmarshall local .firebaserc")
		return ""
	}

	return rc.Projects.Default
}

func getDefaultProject() string {
	var p string

	if p = getFirebaseRC(); p != "" {
		return p
	}

	if p = os.Getenv("GOOGLE_PROJECT_ID"); p != "" {
		return p
	}

	return ""
}

func (p *Params) getArgs() {
	flag.StringVar(&p.LOCATION, "location", "us-central1", "Where do you want to deploy the functions created for this extension?  You usually want a location close to your database. For help selecting a location, refer to the [location selection guide](https://firebase.google.com/docs/functions/locations).")
	flag.StringVar(&p.DATASET_LOCATION, "dataset-location", "us", "Where do you want to deploy the BigQuery dataset created for this extension? For help selecting a location, refer to the [location selection guide](https://cloud.google.com/bigquery/docs/locations).")
	flag.StringVar(&p.BIGQUERY_PROJECT_ID, "project-id", getDefaultProject(), "The GCP Project ID")
	flag.BoolVar(&p.WILDCARD_IDS, "-wildcard-ids", false, "Creates a column containing a JSON object of all wildcard ids from a documents path.")
	flag.StringVar(&p.DATASET_ID, "dataset-id", fmt.Sprintf("%s_firestore_export", strings.Replace(getDefaultProject(), "-", "_", -1)), "The ID of the BigQuery dataset")
	flag.StringVar(&p.TABLE_PARTITIONING, "table-partitioning", "NONE", "This parameter will allow you to partition the BigQuery table and BigQuery view created by the extension based on data ingestion time. You may select the granularity of partitioning based upon one of: HOUR, DAY, MONTH, YEAR. This will	generate one partition per day, hour, month or year, respectively.")
	flag.StringVar(&p.TIME_PARTITIONING_FIELD, "time-partitioning-field", "", "BigQuery table column/schema field name for TimePartitioning. You can choose schema available as `timestamp` OR new custom defined column that will be assigned to the selected Firestore Document field below. Defaults to pseudo column _PARTITIONTIME if unspecified. Cannot be changed if Table is already partitioned.")
	flag.StringVar(&p.TIME_PARTITIONING_FIRESTORE_FIELD, "time-partitioning-firestore-field", "", "This parameter will allow you to partition the BigQuery table  created by the extension based on selected. The Firestore Document field value must be a top-level TIMESTAMP, DATETIME, DATE field BigQuery string format or Firestore timestamp(will be converted to BigQuery TIMESTAMP). Cannot be changed if Table is already partitioned. example: `postDate`")
	flag.StringVar(&p.TIME_PARTITIONING_FIELD_TYPE, "time-partitioning-field-type", "omit", "Parameter for BigQuery SQL schema field type for the selected Time Partitioning Firestore Document field option. Cannot be changed if Table is already partitioned.")
	flag.StringVar(&p.CLUSTERING, "clustering", "", "This parameter will allow you to set up Clustering for the BigQuery Table created by the extension. (for example: `data,document_id,timestamp`- no whitespaces). You can select up to 4 comma separated fields(order matters). Available schema extensions table fields for clustering: `document_id, timestamp, event_id, operation, data`.")
	flag.StringVar(&p.BACKUP_COLLECTION, "backup-collection", "", "This (optional) parameter will allow you to specify a collection for which failed BigQuery updates will be written to.")
	flag.StringVar(&p.TRANSFORM_FUNCTION, "transform-function-url", "", "Specify a function URL to call that will transform the payload that will be written to BigQuery. See the pre-install documentation for more details.")

	flag.BoolVar(&recursive, "-recursive", false, "When -include == 'ALL', -recursive will crawl Firestore to include all subcollections")
	flag.StringVar(&include, "include", "", "A comma separated list of collection names to include. Set to 'ALL' to include all collections")
	flag.StringVar(&exclude, "exclude", "", "A comma separated list of collection names to exclude when include is set to 'ALL'")
	flag.IntVar(&maxWorkers, "max-workers", 5, "The maximum number of works to run in parallel when deploying the extension")

	flag.Parse()
}

func init() {
	tmpDir = fmt.Sprintf("%s/.config/gcloud/logs/fs-to-bq-installer", os.Getenv("HOME"))
}
