package bitbucket

import (
	"encoding/json"
	"fmt"
	"strings"

	embed "github.com/Clinet/discordgo-embed"
	"github.com/bwmarrin/discordgo"
	"github.com/infiniteloopcloud/discord-bitbucket/env"
)

const (
	success   = 0x90EE90
	failure   = 0xD10000
	prCreated = 0x89CFF0
	prUpdated = 0x0047AB
	gray      = 0x979797
)

func Handle(eventType string, body []byte) (string, *discordgo.MessageEmbed, error) {
	switch eventType {
	case "repo:push":
		return handlePush(body)
	case "repo:commit_status_updated":
		return commitStatusUpdated(body)
	case "pullrequest:created":
		return pullRequestCreated(body)
	case "pullrequest:updated":
		return pullRequestUpdated(body)
	case "pullrequest:approved":
		return pullRequestApproved(body)
	case "pullrequest:unapproved":
		return pullRequestUnapproved(body)
	case "pullrequest:fulfilled":
		return pullRequestFulfilled(body)
	case "pullrequest:rejected":
		return pullRequestRejected(body)
	case "pullrequest:comment_created":
		return pullRequestCommentCreated(body)
	case "pullrequest:comment_updated":
		return pullRequestCommentUpdated(body)
	case "pullrequest:comment_deleted":
		return pullRequestCommentDeleted(body)
	}
	return "", nil, nil
}

func handlePush(body []byte) (string, *discordgo.MessageEmbed, error) {
	if env.Configuration().SkipRepoPushMessages {
		return "", nil, nil
	}

	var push RepoPushEvent
	err := json.Unmarshal(body, &push)
	if err != nil {
		return "", nil, err
	}
	numOfCommits := 0
	resourceName := "unknown"
	resourceType := "unknown"
	if len(push.Push.Changes) > 0 {
		numOfCommits = len(push.Push.Changes[0].Commits)
		resourceName = push.Push.Changes[0].New.Name
		resourceType = push.Push.Changes[0].New.Type
	}

	message := embed.NewEmbed().
		SetTitle("Push happened").
		AddField("Number of commits", fmt.Sprintf("%d", numOfCommits)).
		AddField("Resource name", resourceName).
		AddField("Resource type", resourceType).
		SetColor(success)
	if push.Actor.DisplayName != "" {
		message = message.SetDescription(push.Actor.DisplayName + " pushed")
	}

	return push.Repository.Name, message.MessageEmbed, nil
}

func commitStatusUpdated(body []byte) (string, *discordgo.MessageEmbed, error) {
	var event RepoCommitStatusUpdatedEvent
	err := json.Unmarshal(body, &event)
	if err != nil {
		return "", nil, err
	}
	color := gray
	if event.CommitStatus.State == "FAILED" {
		color = failure
	} else if event.CommitStatus.State == "SUCCESSFUL" {
		color = success
	}

	if event.CommitStatus.Name == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(event.CommitStatus.Commit.Author.User.DisplayName, event.CommitStatus.Commit.Author.User.Links.Avatar.Href).
		SetTitle("[" + event.Repository.FullName + "]: " + event.CommitStatus.Name).
		SetColor(color)

	if event.CommitStatus.State != "" {
		message = message.AddField("Status", event.CommitStatus.State)
	}
	if event.CommitStatus.URL != "" {
		message = message.SetURL(event.CommitStatus.URL)
	}

	return event.Repository.Name, message.MessageEmbed, nil
}

func pullRequestCreated(body []byte) (string, *discordgo.MessageEmbed, error) {
	var created PullRequestCreatedEvent
	err := json.Unmarshal(body, &created)
	if err != nil {
		return "", nil, err
	}
	reviewers := "none"
	reviewerList := []string{}
	for _, reviewer := range created.PullRequest.Reviewers {
		reviewerList = append(reviewerList, reviewer.DisplayName)
	}
	if len(reviewerList) > 0 {
		reviewers = strings.Join(reviewerList, ", ")
	}

	if created.Actor.DisplayName == "" || created.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(created.Actor.DisplayName, created.Actor.Links.Avatar.Href).
		SetTitle("["+created.PullRequest.Source.Repository.FullName+"]:"+" Pull request opened: "+created.PullRequest.Title).
		SetColor(prCreated).
		AddField("Reviewers", reviewers)

	if created.PullRequest.Source.Branch.Name != "" && created.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + created.PullRequest.Source.Branch.Name + "` > `" + created.PullRequest.Destination.Branch.Name + "`")
	}
	if created.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(created.PullRequest.Links.HTML.Href)
	}
	if created.PullRequest.State != "" {
		message = message.AddField("Status", created.PullRequest.State)
	}
	if created.PullRequest.Description != "" {
		if len(created.PullRequest.Description) > 200 {
			desc := created.PullRequest.Description[0:199] + "..."
			message = message.AddField("PR Description", "**"+desc+"**")
		} else {
			message = message.AddField("PR Description", "**"+created.PullRequest.Description+"**")
		}
	}

	return created.Repository.Name, message.MessageEmbed, nil
}

func pullRequestUpdated(body []byte) (string, *discordgo.MessageEmbed, error) {
	var updated PullRequestUpdatedEvent
	err := json.Unmarshal(body, &updated)
	if err != nil {
		return "", nil, err
	}
	reviewers := "none"
	reviewerList := []string{}
	for _, reviewer := range updated.PullRequest.Participants {
		if reviewer.Approved {
			reviewerList = append(reviewerList, "**✓**"+reviewer.User.DisplayName)
		} else {
			reviewerList = append(reviewerList, "**x **"+reviewer.User.DisplayName)
		}
	}
	if len(reviewerList) > 0 {
		reviewers = strings.Join(reviewerList, "\n")
	}

	if updated.Actor.DisplayName == "" || updated.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(updated.Actor.DisplayName, updated.Actor.Links.Avatar.Href).
		SetTitle("["+updated.PullRequest.Source.Repository.FullName+"]:"+" Pull request updated: "+updated.PullRequest.Title).
		SetColor(prUpdated).
		AddField("Reviewers", reviewers)

	if updated.PullRequest.Source.Branch.Name != "" && updated.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + updated.PullRequest.Source.Branch.Name + "` > `" + updated.PullRequest.Destination.Branch.Name + "`")
	}
	if updated.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(updated.PullRequest.Links.HTML.Href)
	}
	if updated.PullRequest.State != "" {
		message = message.AddField("Status", updated.PullRequest.State)
	}
	if updated.PullRequest.Description != "" {
		if len(updated.PullRequest.Description) > 200 {
			desc := updated.PullRequest.Description[0:199] + "..."
			message = message.AddField("PR Description", "**"+desc+"**")
		} else {
			message = message.AddField("PR Description", "**"+updated.PullRequest.Description+"**")
		}
	}

	return updated.Repository.Name, message.MessageEmbed, nil
}

func pullRequestApproved(body []byte) (string, *discordgo.MessageEmbed, error) {
	var approved PullRequestApprovedEvent
	err := json.Unmarshal(body, &approved)
	if err != nil {
		return "", nil, err
	}

	if approved.Actor.DisplayName == "" || approved.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(approved.Approval.User.DisplayName, approved.Approval.User.Links.Avatar.Href).
		SetTitle("[" + approved.PullRequest.Source.Repository.FullName + "]:" + " Pull request approved: " + approved.PullRequest.Title).
		SetColor(success)

	if approved.PullRequest.Source.Branch.Name != "" && approved.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + approved.PullRequest.Source.Branch.Name + "` > `" + approved.PullRequest.Destination.Branch.Name + "`")
	}
	if approved.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(approved.PullRequest.Links.HTML.Href)
	}
	if approved.Actor.DisplayName != "" {
		message = message.AddField("Created by", approved.PullRequest.Author.DisplayName)
	}

	return approved.Repository.Name, message.MessageEmbed, nil
}

func pullRequestUnapproved(body []byte) (string, *discordgo.MessageEmbed, error) {
	var unapproved PullRequestApprovedEvent
	err := json.Unmarshal(body, &unapproved)
	if err != nil {
		return "", nil, err
	}

	if unapproved.Actor.DisplayName == "" || unapproved.PullRequest.Title == "" {
		return "", nil, nil
	}

	//message := embed.NewEmbed().SetTitle(unapproved.Approval.User.DisplayName + " unapproved pull request: " + unapproved.PullRequest.Title).SetColor(success)

	message := embed.NewEmbed().
		SetAuthor(unapproved.Approval.User.DisplayName, unapproved.Approval.User.Links.Avatar.Href).
		SetTitle("[" + unapproved.PullRequest.Source.Repository.FullName + "]:" + " Pull request unapproved: " + unapproved.PullRequest.Title).
		SetColor(failure)

	if unapproved.PullRequest.Source.Branch.Name != "" && unapproved.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + unapproved.PullRequest.Source.Branch.Name + "` > `" + unapproved.PullRequest.Destination.Branch.Name + "`")
	}
	if unapproved.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(unapproved.PullRequest.Links.HTML.Href)
	}
	if unapproved.Actor.DisplayName != "" {
		message = message.AddField("Created by", unapproved.PullRequest.Author.DisplayName)
	}

	return unapproved.Repository.Name, message.MessageEmbed, nil
}

func pullRequestFulfilled(body []byte) (string, *discordgo.MessageEmbed, error) {
	var merged PullRequestMergedEvent
	err := json.Unmarshal(body, &merged)
	if err != nil {
		return "", nil, err
	}
	reviewers := "none"
	reviewerList := []string{}
	for _, reviewer := range merged.PullRequest.Participants {
		if reviewer.Approved {
			reviewerList = append(reviewerList, "**✓**"+reviewer.User.DisplayName)
		} else {
			reviewerList = append(reviewerList, "**x **"+reviewer.User.DisplayName)
		}
	}
	if len(reviewerList) > 0 {
		reviewers = strings.Join(reviewerList, "\n")
	}

	if merged.PullRequest.ClosedBy.DisplayName == "" || merged.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(merged.Actor.DisplayName, merged.PullRequest.ClosedBy.Links.Avatar.Href).
		SetTitle("["+merged.Repository.FullName+"]: Pull request merged: "+merged.PullRequest.Title).
		SetColor(success).
		AddField("Reviewers", reviewers)

	if merged.PullRequest.Source.Branch.Name != "" && merged.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + merged.PullRequest.Source.Branch.Name + "` > `" + merged.PullRequest.Destination.Branch.Name + "`")
	}
	if merged.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(merged.PullRequest.Links.HTML.Href)
	}
	if merged.Actor.DisplayName != "" {
		message = message.AddField("Created by", merged.Actor.DisplayName)
	}
	if merged.PullRequest.State != "" {
		message = message.AddField("Status", merged.PullRequest.State)
	}

	return merged.Repository.Name, message.MessageEmbed, nil
}

func pullRequestRejected(body []byte) (string, *discordgo.MessageEmbed, error) {
	var rejected PullRequestMergedEvent
	err := json.Unmarshal(body, &rejected)
	if err != nil {
		return "", nil, err
	}

	if rejected.PullRequest.ClosedBy.DisplayName == "" || rejected.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(rejected.Actor.DisplayName, rejected.PullRequest.ClosedBy.Links.Avatar.Href).
		SetTitle("[" + rejected.Repository.FullName + "]: Pull request rejected: " + rejected.PullRequest.Title).
		SetColor(failure)

	if rejected.PullRequest.Source.Branch.Name != "" && rejected.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + rejected.PullRequest.Source.Branch.Name + "` > `" + rejected.PullRequest.Destination.Branch.Name + "`")
	}
	if rejected.PullRequest.Links.HTML.Href != "" {
		message = message.SetURL(rejected.PullRequest.Links.HTML.Href)
	}
	if rejected.Actor.DisplayName != "" {
		message = message.AddField("Created by", rejected.Actor.DisplayName)
	}
	if rejected.PullRequest.State != "" {
		message = message.AddField("Status", rejected.PullRequest.State)
	}

	return rejected.Repository.Name, message.MessageEmbed, nil
}

func pullRequestCommentCreated(body []byte) (string, *discordgo.MessageEmbed, error) {
	var commentCreated PullRequestCommentCreatedEvent
	err := json.Unmarshal(body, &commentCreated)
	if err != nil {
		return "", nil, err
	}

	comment := "no comment"
	if commentCreated.Comment.Content.Raw != "" {
		comment = commentCreated.Comment.Content.Raw
		if len(comment) > 105 {
			comment = commentCreated.Comment.Content.Raw[0:100] + "..."
		}
	}

	if commentCreated.Comment.User.DisplayName == "" || commentCreated.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(commentCreated.Actor.DisplayName, commentCreated.Actor.Links.Avatar.Href).
		SetTitle("["+commentCreated.PullRequest.Destination.Repository.FullName+"]:"+" Comment created on pull request: "+commentCreated.PullRequest.Title).
		AddField("Comment", comment).
		SetColor(prCreated)

	if commentCreated.PullRequest.Source.Branch.Name != "" && commentCreated.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + commentCreated.PullRequest.Source.Branch.Name + "` > `" + commentCreated.PullRequest.Destination.Branch.Name + "`")
	}
	if commentCreated.Comment.Links.HTML.Href != "" {
		message = message.SetURL(commentCreated.Comment.Links.HTML.Href)
	}

	return commentCreated.Repository.Name, message.MessageEmbed, nil
}

func pullRequestCommentUpdated(body []byte) (string, *discordgo.MessageEmbed, error) {
	var commentUpdated PullRequestCommentCreatedEvent
	err := json.Unmarshal(body, &commentUpdated)
	if err != nil {
		return "", nil, err
	}

	if commentUpdated.Comment.User.DisplayName == "" || commentUpdated.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(commentUpdated.Actor.DisplayName, commentUpdated.Actor.Links.Avatar.Href).
		SetTitle("["+commentUpdated.PullRequest.Destination.Repository.FullName+"]:"+" Comment updated on pull request: "+commentUpdated.PullRequest.Title).
		AddField("Author:", commentUpdated.Comment.User.DisplayName).
		SetColor(prUpdated)

	if commentUpdated.PullRequest.Source.Branch.Name != "" && commentUpdated.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + commentUpdated.PullRequest.Source.Branch.Name + "` > `" + commentUpdated.PullRequest.Destination.Branch.Name + "`")
	}
	if commentUpdated.Comment.Links.HTML.Href != "" {
		message = message.SetURL(commentUpdated.Comment.Links.HTML.Href)
	}

	return commentUpdated.Repository.Name, message.MessageEmbed, nil
}

func pullRequestCommentDeleted(body []byte) (string, *discordgo.MessageEmbed, error) {
	var commentDeleted PullRequestCommentCreatedEvent
	err := json.Unmarshal(body, &commentDeleted)
	if err != nil {
		return "", nil, err
	}

	if commentDeleted.Comment.User.DisplayName == "" || commentDeleted.PullRequest.Title == "" {
		return "", nil, nil
	}

	message := embed.NewEmbed().
		SetAuthor(commentDeleted.Actor.DisplayName, commentDeleted.Actor.Links.Avatar.Href).
		SetTitle("["+commentDeleted.PullRequest.Destination.Repository.FullName+"]:"+" Comment deleted on pull request: "+commentDeleted.PullRequest.Title).
		AddField("Author:", commentDeleted.Comment.User.DisplayName).
		SetColor(failure)

	if commentDeleted.PullRequest.Source.Branch.Name != "" && commentDeleted.PullRequest.Destination.Branch.Name != "" {
		message = message.SetDescription("`" + commentDeleted.PullRequest.Source.Branch.Name + "` > `" + commentDeleted.PullRequest.Destination.Branch.Name + "`")
	}
	if commentDeleted.Comment.Links.HTML.Href != "" {
		message = message.SetURL(commentDeleted.Comment.Links.HTML.Href)
	}

	return commentDeleted.Repository.Name, message.MessageEmbed, nil
}
