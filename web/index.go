package web

import (
	"TUM-Live/dao"
	"TUM-Live/model"
	"TUM-Live/tools"
	"TUM-Live/tools/tum"
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strconv"
)

var VersionTag string

func MainPage(c *gin.Context) {
	tName := sentry.TransactionName("GET /")
	spanMain := sentry.StartSpan(c.Request.Context(), "MainPageHandler", tName)
	defer spanMain.Finish()
	res, err := dao.AreUsersEmpty(context.Background()) // fresh installation?
	if err != nil {
		_ = templ.ExecuteTemplate(c.Writer, "error.gohtml", nil)
		return
	} else if res {
		_ = templ.ExecuteTemplate(c.Writer, "onboarding.gohtml", nil)
		return
	}
	indexData := NewIndexData()
	var tumLiveContext tools.TUMLiveContext
	tumLiveContextQueried, found := c.Get("TUMLiveContext")
	if found {
		tumLiveContext = tumLiveContextQueried.(tools.TUMLiveContext)
		indexData.TUMLiveContext = tumLiveContext
	} else { //TODO log
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var year int
	var term string
	if c.Param("year") == "" {
		year, term = tum.GetCurrentSemester()
	} else {
		term = c.Param("term")
		year, err = strconv.Atoi(c.Param("year"))
		if err != nil || (term != "W" && term != "S") {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Bad semester format in url."})
			return
		}
	}
	indexData.Semesters = dao.GetAvailableSemesters(spanMain.Context())
	indexData.CurrentYear = year
	indexData.CurrentTerm = term
	if tumLiveContext.User != nil && tumLiveContext.User.Role == model.AdminType {
		indexData.Courses = dao.GetAllCoursesForSemester(year, term, spanMain.Context())
	} else if tumLiveContext.User != nil {
		indexData.Courses = tumLiveContext.User.CoursesForSemester(year, term, spanMain.Context())
	}
	streams, err := dao.GetCurrentLive(context.Background())
	var livestreams []CourseStream

	for _, stream := range streams {
		courseForLiveStream, _ := dao.GetCourseById(context.Background(), stream.CourseID)
		// Todo: refactor into dao
		if courseForLiveStream.Visibility == "loggedin" && tumLiveContext.User == nil {
			continue
		}
		if courseForLiveStream.Visibility == "enrolled" {
			if !dao.IsUserAllowedToWatchPrivateCourse(courseForLiveStream, tumLiveContext.User) {
				continue
			}
		}
		livestreams = append(livestreams, CourseStream{
			Course: courseForLiveStream,
			Stream: stream,
		})
	}
	indexData.LiveStreams = livestreams
	public, err := dao.GetPublicCourses(year, term)
	if err != nil {
		indexData.PublicCourses = []model.Course{}
	} else {
		// filter out courses that already are in "my courses"
		var publicFiltered []model.Course
		for _, c := range public {
			if !tools.CourseListContains(indexData.Courses, c.ID) {
				publicFiltered = append(publicFiltered, c)
			}
		}
		if tumLiveContext.User != nil {
			loggedIn, _ := dao.GetCoursesForLoggedInUsers(year, term)
			for _, c := range loggedIn {
				if !tools.CourseListContains(indexData.Courses, c.ID) {
					publicFiltered = append(publicFiltered, c)
				}
			}
		}
		indexData.PublicCourses = publicFiltered
	}
	sort.Slice(indexData.PublicCourses, func(i, j int) bool {
		return indexData.PublicCourses[i].CompareTo(indexData.PublicCourses[j])
	})
	sort.Slice(indexData.Courses, func(i, j int) bool {
		return indexData.Courses[i].CompareTo(indexData.Courses[j])
	})
	_ = templ.ExecuteTemplate(c.Writer, "index.gohtml", indexData)
}

func AboutPage(c *gin.Context) {
	var indexData IndexData
	var tumLiveContext tools.TUMLiveContext
	tumLiveContextQueried, found := c.Get("TUMLiveContext")
	if found {
		tumLiveContext = tumLiveContextQueried.(tools.TUMLiveContext)
		indexData.TUMLiveContext = tumLiveContext
	} else { //TODO log
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	indexData.VersionTag = VersionTag

	_ = templ.ExecuteTemplate(c.Writer, "about.gohtml", indexData)
}

type IndexData struct {
	VersionTag     string
	TUMLiveContext tools.TUMLiveContext
	IsUser         bool
	IsAdmin        bool
	IsStudent      bool
	LiveStreams    []CourseStream
	Courses        []model.Course
	PublicCourses  []model.Course
	Semesters      []dao.Semester
	CurrentYear    int
	CurrentTerm    string
	UserName       string
}

func NewIndexData() IndexData {
	return IndexData{
		VersionTag: VersionTag,
	}
}

type CourseStream struct {
	Course model.Course
	Stream model.Stream
}
