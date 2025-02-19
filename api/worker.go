package api

import (
	"TUM-Live/dao"
	"TUM-Live/model"
	"TUM-Live/tools"
	"context"
	"encoding/json"
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

func configGinWorkerRouter(r *gin.Engine) {
	workers := r.Group("/api/worker")
	workers.Use(tools.Worker)
	r.GET("/api/worker/getJobs/:workerID", getJob)
	r.POST("/api/worker/putVOD/:workerID", putVod)
	r.POST("/api/worker/ping/:workerID", ping)
	r.POST("/api/worker/notifyLive/:workerID", notifyLive)
	r.POST("/api/worker/notifyLiveEnd/:workerID/:streamID", notifyLiveEnd)
	workers.POST("/silenceResults/:workerID", silenceResults)
}

type SilenceReq struct {
	StreamID string          `json:"stream_id"`
	Silences []model.Silence `json:"silences"`
}

func silenceResults(c *gin.Context) {
	var req SilenceReq
	err := c.MustBindWith(&req, binding.JSON)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if s, err := dao.GetStreamByID(c, req.StreamID); err == nil {
		s.Silences = req.Silences
		for i, _ := range req.Silences {
			req.Silences[i].StreamID = s.ID
		}
		err = dao.UpdateSilences(req.Silences, req.StreamID)
		if err != nil {
			log.Printf("%v", err)
			sentry.CaptureException(err)
			return
		}
	}
}

func ping(c *gin.Context) {
	if worker, err := dao.GetWorkerByID(context.Background(), c.Param("workerID")); err == nil {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Couldn't read ping request")
			sentry.CaptureException(err)
			return
		}
		var req pingReq
		err = json.Unmarshal(body, &req)
		if err != nil {
			log.Printf("Couldn't unmarshal ping request")
			sentry.CaptureException(err)
			return
		}
		worker.Workload = req.Workload
		worker.Status = req.Status
		worker.LastSeen = time.Now()
		dao.SaveWorker(worker)
	}
}

type pingReq struct {
	Workload int    `json:"workload,omitempty"`
	Status   string `json:"status,omitempty"`
}

func notifyLiveEnd(c *gin.Context) {
	_, err := dao.GetWorkerByID(context.Background(), c.Param("workerID"))
	if err != nil {
		c.JSON(http.StatusForbidden, "forbidden")
		return
	}
	if sid := c.Param("streamID"); sid != "" {
		err := dao.SetStreamNotLiveById(sid)
		if err != nil {
			log.Printf("Couldn't set stream not live: %v\n", err)
			sentry.CaptureException(err)
			return
		}
		return

	}
	c.JSON(http.StatusNotFound, "forbidden")
	return

}
func notifyLive(c *gin.Context) {
	_, err := dao.GetWorkerByID(context.Background(), c.Param("workerID"))
	if err != nil {
		c.JSON(http.StatusForbidden, "forbidden")
		return
	}
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var req notifyLiveRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	stream, err := dao.GetStreamByID(context.Background(), req.StreamID)
	if err != nil {
		sentry.CaptureException(err)
		return
	}
	stream.LiveNow = true
	switch req.Version {
	case "COMB":
		stream.PlaylistUrl = req.URL
	case "PRES":
		stream.PlaylistUrlPRES = req.URL
	case "CAM":
		stream.PlaylistUrlCAM = req.URL
	}
	err = dao.SaveStream(&stream)
	if err != nil {
		sentry.CaptureException(err)
	}
}

type notifyLiveRequest struct {
	StreamID string `json:"streamID"`
	URL      string `json:"url"`     // eg. https://live.lrz.de/livetum/stream/playlist.m3u8
	Version  string `json:"version"` //eg. COMB
}

func putVod(c *gin.Context) {
	_, err := dao.GetWorkerByID(context.Background(), c.Param("workerID"))
	if err != nil {
		c.JSON(http.StatusForbidden, "forbidden")
		return
	}
	jsondata, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var req putVodData
	err = json.Unmarshal(jsondata, &req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	stream, _ := dao.GetStreamByID(context.Background(), req.StreamId)
	stream.Recording = true
	switch req.Version {
	case "COMB":
		stream.PlaylistUrl = req.HlsUrl
	case "PRES":
		stream.PlaylistUrlPRES = req.HlsUrl
	case "CAM":
		stream.PlaylistUrlCAM = req.HlsUrl
	default:
		sentry.CaptureException(errors.New("invalid source type: " + req.Version))
	}
	stream.FilePath = req.FilePath
	_ = dao.SaveStream(&stream)
}

func getJob(c *gin.Context) {
	_, err := dao.GetWorkerByID(context.Background(), c.Param("workerID"))
	if err != nil {
		c.JSON(http.StatusForbidden, "forbidden")
		return
	}
	job, err := dao.PickJob(context.Background())
	if err != nil {
		c.JSON(http.StatusNotFound, &jobData{})
		return
	}
	stream, _ := dao.GetStreamByID(context.Background(), strconv.Itoa(int(job.StreamID)))
	course, _ := dao.GetCourseById(context.Background(), stream.CourseID)
	c.JSON(http.StatusOK, &jobData{
		Id:          job.ID,
		Name:        stream.Name,
		StreamStart: stream.Start,
		StreamId:    job.StreamID,
		Path:        job.FilePath,
		Upload:      course.VODEnabled,
	})
}

type jobData struct {
	Id          uint      `json:"id"`
	Name        string    `json:"name"`
	StreamId    uint      `json:"streamId"`
	StreamStart time.Time `json:"streamStart"`
	Path        string    `json:"path"`
	Upload      bool
}

type putVodData struct {
	HlsUrl   string
	Version  string
	FilePath string
	StreamId string
}
