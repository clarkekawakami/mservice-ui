package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	pb "github.com/gaterace/mservice/pkg/mserviceaccount"
	"github.com/gin-gonic/gin"
)

// *********************************
// local funcs to get mservice data
// *********************************
func getClaimNames(client pb.MServiceAccountClient, mctx context.Context) *pb.GetClaimNamesResponse {
	fmt.Println("in getClaimNames")
	req := pb.GetClaimNamesRequest{}
	req.DummyParam = 1
	resp, err := client.GetClaimNames(mctx, &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	return resp

}

func getClaimNameById(client pb.MServiceAccountClient, id int64, mctx context.Context) *pb.GetClaimNameByIdResponse {
	fmt.Println("in getClaimNameById")
	req := pb.GetClaimNameByIdRequest{}
	req.ClaimNameId = id
	resp, err := client.GetClaimNameById(mctx, &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	return resp

}

func getClaimValuesByNameId(client pb.MServiceAccountClient, id int64, mctx context.Context) *pb.GetClaimValuesByNameIdResponse {
	fmt.Println("in getClaimValuesByNameId")
	req := pb.GetClaimValuesByNameIdRequest{}
	req.ClaimNameId = id
	resp, err := client.GetClaimValuesByNameId(mctx, &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	return resp

}

func getClaimValueById(client pb.MServiceAccountClient, id int64, mctx context.Context) *pb.GetClaimValueByIdResponse {
	fmt.Println("in getClaimValueById")
	req := pb.GetClaimValueByIdRequest{}
	req.ClaimValueId = id
	resp, err := client.GetClaimValueById(mctx, &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	return resp
}

// ***********************************************
// public funcs to send back htmx fragments/pages
// ***********************************************

// get Claim data... target sub_app_content
func GetSubappClaims(c *gin.Context) {

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getClaimNames(client, mctx)

	c.HTML(http.StatusOK,
		"subapp_partial.html",
		gin.H{
			"auth":        c.GetBool("auth"),
			"claims":      c.GetStringMap("claims"),
			"sub_title":   "Mservice Claims",
			"sub_active":  "claims",
			"claim_names": resp,
		},
	)

}

// Get ClaimName and display claim name form - hx-target="#action_content"
func GetClaimById(c *gin.Context) {

	if c.Param("id") != "new" {
		claimId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		conn := getGrpcConn()
		defer conn.Close()

		client := pb.NewMServiceAccountClient(conn)

		mctx := getMctxFromCookie(c)

		resp := getClaimNameById(client, claimId, mctx)

		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
				"claim_name": resp,
			},
		)

	} else { // new claim name rec

		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"auth":     c.GetBool("auth"),
				"claims":   c.GetStringMap("claims"),
				"isCreate": true,
			},
		)

	}

}

// Create new ClaimName and display claims list - hx-target="#claims_list"
func CreateClaim(c *gin.Context) {

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.CreateClaimNameRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.CreateClaimName(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("create claim error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"claim":   req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// reget the claim names and render claims list

		resp := getClaimNames(client, mctx)

		c.HTML(http.StatusOK,
			"claims_list.html",
			gin.H{
				"auth":        c.GetBool("auth"),
				"claims":      c.GetStringMap("claims"),
				"claim_names": resp,
			},
		)

	}

}

// Update ClaimName and display claim name form - hx-target="#action_content"
func UpdateClaim(c *gin.Context) {

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.UpdateClaimNameRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.UpdateClaimName(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update claim error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"claim":   req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {

		resp1 := getClaimNameById(client, req.ClaimNameId, mctx)

		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
				"claim_name": resp1,
				"success":    true,
			},
		)
	}

}

// delete claim name and display claims list target #action_content
func DeleteClaim(c *gin.Context) {
	claimId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// var version int32
	version, _ := strconv.ParseInt(c.Param("version"), 10, 32)

	fmt.Println("id:", claimId)
	fmt.Println("version:", version)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// Delete the record
	req := pb.DeleteClaimNameRequest{}
	req.ClaimNameId = claimId
	req.Version = int32(version)
	resp, err := client.DeleteClaimName(mctx, &req)
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}

	// then get working actname users

	resp1 := getClaimNames(client, mctx)

	c.HTML(http.StatusOK,
		"claims.html",
		gin.H{
			"auth":        c.GetBool("auth"),
			"claims":      c.GetStringMap("claims"),
			"claim_names": resp1,
		},
	)

}

// get claim values then render claim_values_list... target #action_content
func GetSubappClaimsValues(c *gin.Context) {

	claimName := c.Request.URL.Query()["claim_name"][0]
	// fmt.Println("claimName param", claimName)
	claimId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// fmt.Println("claimId param", claimId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getClaimValuesByNameId(client, claimId, mctx)

	c.HTML(http.StatusOK,
		"claim_values_list.html",
		gin.H{
			"claim_values": resp,
			"for_claim":    claimName,
			"claim_id":     claimId,
		},
	)

}

// get claim value (if not new) then render claim_value_form... target #action_content
func GetClaimValueById(c *gin.Context) {
	var claim_id int64

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	if c.Param("id") != "new" {
		claimValueId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		resp := getClaimValueById(client, claimValueId, mctx)

		c.HTML(http.StatusOK,
			"claim_value_form.html",
			gin.H{
				"auth":        c.GetBool("auth"),
				"claims":      c.GetStringMap("claims"),
				"claim_value": resp,
			},
		)

	} else { // new claim name rec

		fmt.Println("raw claim param............", c.Request.URL.Query()["claim_id"][0])

		i, err := strconv.ParseInt(c.Request.URL.Query()["claim_id"][0], 10, 64)
		if err != nil {
			panic(err)
		} else {
			claim_id = i
		}
		fmt.Println("working claim_id for new:::", claim_id)

		for_claim := c.Request.URL.Query()["claim_name"][0]
		fmt.Println("for claim new:::", for_claim)

		c.HTML(http.StatusOK,
			"claim_value_form.html",
			gin.H{
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
				"isCreate":   true,
				"claim_id":   claim_id,
				"claim_name": for_claim,
			},
		)

	}

}

// Create new Claim value and display claim value list - hx-target="#action_content"
func CreateClaimValue(c *gin.Context) {
	fmt.Println("*********** in create claim value ***********")

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.CreateClaimValueRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.CreateClaimValue(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("create claim val error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"claim_value_form.html",
			gin.H{
				"claim":   req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// create worked... go back to claim value list

		req1 := pb.GetClaimValuesByNameIdRequest{}
		req1.ClaimNameId = req.ClaimNameId
		resp1 := getClaimValuesByNameId(client, req.ClaimNameId, mctx)

		// fmt.Println("claim name????", resp1.ClaimValue[0].Claim.ClaimName)

		c.HTML(http.StatusOK,
			"claim_values_list.html",
			gin.H{
				"auth":         c.GetBool("auth"),
				"claims":       c.GetStringMap("claims"),
				"claim_values": resp1,
				"for_claim":    resp1.ClaimValue[0].Claim.ClaimName,
				"claim_id":     resp1.ClaimValue[0].ClaimNameId,
			},
		)

	}

}

// Update ClaimName and display claim name form - hx-target="#action_content"
func UpdateClaimValue(c *gin.Context) {

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.UpdateClaimValueRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.UpdateClaimValue(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update claim error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"claim_name_form.html",
			gin.H{
				"claim":   req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// req1 := pb.GetClaimValueByIdRequest{}
		// req1.ClaimValueId = req.ClaimValueId
		resp1 := getClaimValueById(client, req.ClaimValueId, mctx)

		c.HTML(http.StatusOK,
			"claim_value_form.html",
			gin.H{
				"auth":        c.GetBool("auth"),
				"claims":      c.GetStringMap("claims"),
				"claim_value": resp1,
				"success":     true,
			},
		)
	}

}

// delete claim value rec, get claim values, render claim_values_list target #action_content
func DeleteClaimValue(c *gin.Context) {
	claimValueId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	claimNameId, _ := strconv.ParseInt(c.Param("claim_id"), 10, 64)
	// var version int32
	version, _ := strconv.ParseInt(c.Param("version"), 10, 32)
	claimName := c.Request.URL.Query()["claim_name"][0]

	fmt.Println("id:", claimValueId)
	fmt.Println("version:", version)
	fmt.Println("for_claim:", claimName)
	fmt.Println("claim_id:", claimNameId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// Delete the record
	req := pb.DeleteClaimValueRequest{}
	req.ClaimValueId = claimValueId
	req.Version = int32(version)
	resp, err := client.DeleteClaimValue(mctx, &req)
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}

	// then get claim values for claim name
	resp1 := getClaimValuesByNameId(client, claimNameId, mctx)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	c.HTML(http.StatusOK,
		"claim_values_list.html",
		gin.H{
			"claim_values": resp1,
			"for_claim":    claimName,
			"claim_id":     claimNameId,
		},
	)

}
