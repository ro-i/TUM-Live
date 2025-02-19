class CreateCourse {
    private courseIDInput: HTMLInputElement
    private loadFromTUMOnlineBtn: HTMLDivElement
    private courseNameInput: HTMLInputElement
    private teachingTermInput: HTMLInputElement;
    private slugInput: HTMLInputElement;
    private tumOnlineInfo: HTMLSpanElement;
    private enrolledRadio: HTMLInputElement;

    constructor() {
        this.loadFromTUMOnlineBtn = document.getElementById("loadCourseInfoBtn") as HTMLDivElement
        this.loadFromTUMOnlineBtn.addEventListener("click", (e: Event) => this.loadCourseInfo())
        this.courseIDInput = document.getElementById("courseID") as HTMLInputElement
        this.courseNameInput = document.getElementById("name") as HTMLInputElement
        this.teachingTermInput = document.getElementById("teachingTerm") as HTMLInputElement
        this.slugInput = document.getElementById("slug") as HTMLInputElement
        this.tumOnlineInfo = document.getElementById("TUMOnlineInfo") as HTMLSpanElement
        this.enrolledRadio = document.getElementById("enrolled") as HTMLInputElement
        document.getElementById("createCourseBtn").addEventListener("click", (e: Event) => this.createCourse())
    }

    private loadCourseInfo(): void {
        postData("/api/courseInfo", {"courseID": this.courseIDInput.value})
            .then(data => {
                if (data.status != 200) {
                    showMessage("The course with this ID was not found in TUMOnline. Please verify the ID or reach out to us.")
                } else {
                    data.text().then(data => {
                        const json = JSON.parse(data)
                        this.courseNameInput.value = json["courseName"]
                        this.teachingTermInput.value = json["teachingTerm"]
                        this.tumOnlineInfo.innerText = "Currently there are " + json["numberAttendees"] + " students enrolled in this course. Please verify that this looks right."
                        this.enrolledRadio.removeAttribute("disabled")
                    })
                }
            })
    }

    private createCourse(): void {
        const f = new FormData(document.getElementById("createCourseForm") as HTMLFormElement)
        postData("/api/createCourse", {
            "courseID": f.get("courseID"),
            "name": f.get("name"),
            "teachingTerm": f.get("teachingTerm"),
            "slug": f.get("slug"),
            "access": f.get("access"),
            "enVOD": f.get("enVOD") === "on",
            "enDL": f.get("enDL") === "on",
            "enChat": f.get("enChat") === "on",
        })
            .then(data => {
                if (data.status != 200) {
                    data.text().then(t => showMessage(t))
                } else {
                    window.location.href = "/admin"
                }
            })
    }
}

new CreateCourse()
