package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"social-net/auth"
	"social-net/comments"
	"social-net/db"
	"social-net/events"
	"social-net/folowers"
	"social-net/groups"
	"social-net/messages"
	"social-net/notification"
	"social-net/posts"
	"social-net/profile"
	"social-net/session"
	"social-net/utils"
)

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	db.Initdb()

	http.HandleFunc("/api/auth/", auth.Auth)
	http.HandleFunc("/middle", session.Middleware)
	http.HandleFunc("/api/info", auth.Getinfo)

	http.HandleFunc("/api/userinfo", profile.GetUserInfo)
	http.HandleFunc("/api/updateprivacy", profile.UpdatePrivacy)
	http.HandleFunc("/api/setprivacy", profile.UpdatePrivacy)
	http.HandleFunc("/api/ownposts", profile.GetOwnPosts)
	http.HandleFunc("/api/isfollowing", profile.IsFollowing)
	http.HandleFunc("/api/followers", folowers.SendJSON)
	http.HandleFunc("/api/getfollowingfolowers", profile.GetFollowersAndFollowing)
	http.HandleFunc("/api/postsprivacy", profile.GetFollowersAndFollowingPosts)
	http.HandleFunc("/api/checkmyprivacy", profile.CheckMyPrivacy)
	http.HandleFunc("/api/getinvitationsfollow", profile.GetInvitationsFollow)
	http.HandleFunc("/api/accepteinvi", profile.AcceptInvitation)

	http.HandleFunc("/api/posts", posts.Post)
	http.HandleFunc("/api/getposts", posts.Getposts)
	http.HandleFunc("/api/getcomments", comments.Getcomments)
	http.HandleFunc("/api/addcomments", comments.AddComments)

	http.HandleFunc("/api/getmessages", messages.GetMessages)
	http.HandleFunc("/api/messages", messages.GetMessages)
	http.HandleFunc("/ws", messages.Handleconnections)
	http.HandleFunc("/api/openchat", messages.OpenChat)

	http.HandleFunc("/api/creategroups", groups.CreateGroup)
	http.HandleFunc("/api/getgroups", groups.GetGroups)
	http.HandleFunc("/api/addmembertogroup", groups.AddMemberToGroup)
	http.HandleFunc("/api/requesttojoingroup", groups.RequestToJoinGroup)
	http.HandleFunc("/api/removememberfromgroup", groups.RemoveMemberFromGroup)
	http.HandleFunc("/api/acceptgroupmember", groups.AcceptGroupMember)
	http.HandleFunc("/api/cancelgrouprequest", groups.CancelGroupRequest)
	http.HandleFunc("/api/mygroups", groups.MyGroups)
	http.HandleFunc("/api/pendinginvitations", groups.GetPendingInvitations)
	http.HandleFunc("/api/handleinvitation", groups.HandleInvitation)
	http.HandleFunc("/api/GetInvitations", groups.GetPendingInvitations)
	http.HandleFunc("/api/ismember", groups.IsGroupMember)
	http.HandleFunc("/api/checkmem", groups.CheckGroupMembershipStatus)
	http.HandleFunc("/api/acceptgroupinvite", groups.HandleInvitation)
	http.HandleFunc("/api/declinegroupinvite", groups.HandleInvitation)
	http.HandleFunc("/api/groupcomments/add", groups.AddGroupComment)
	http.HandleFunc("/api/groupcomments", groups.GetGroupComments)
	http.HandleFunc("/api/user/pendinginvites", groups.GetUserPendingInvitations)
	http.HandleFunc("/api/groupmembers/status", groups.GetGroupMemberStatuses)

	http.HandleFunc("/api/groupposts", groups.GetGroupPosts)
	http.HandleFunc("/api/groupposts/add", groups.AddGroupPost)

	http.HandleFunc("/api/postsprv", posts.PostPrivacy)
	http.HandleFunc("/api/events", events.GetEvents)
	http.HandleFunc("/api/events/add", events.CreateEvent)
	http.HandleFunc("/api/notifications", notification.GetNotifications)
	http.HandleFunc("/api/markasread", notification.MarkNotificationAsRead)
	http.HandleFunc("/api/events/join", events.JoinEvent)

	http.HandleFunc("/ws/group/", messages.HandleGroupWebSocket)
	http.HandleFunc("/ws/notifications", notification.HandleNotificationWebSocket)

	http.HandleFunc("/api/allusers", utils.Users)
	http.HandleFunc("/api/getavatar", auth.GetAvatar)

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {

			db.DB.Close()
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Println("signal err:", sig)
	db.DB.Close()

}
