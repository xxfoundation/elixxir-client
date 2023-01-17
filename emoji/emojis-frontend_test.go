package emoji

type EmojiMartData struct {
	Categories []struct {
		Id     string   `json:"id"`
		Emojis []string `json:"emojis"`
	} `json:"categories"`
	Emojis struct {
		Field1 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"100"`
		Field2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"1234"`
		Grinning struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grinning"`
		Smiley struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smiley"`
		Smile struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smile"`
		Grin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grin"`
		Laughing struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"laughing"`
		SweatSmile struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sweat_smile"`
		RollingOnTheFloorLaughing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rolling_on_the_floor_laughing"`
		Joy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"joy"`
		SlightlySmilingFace struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"slightly_smiling_face"`
		UpsideDownFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"upside_down_face"`
		MeltingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"melting_face"`
		Wink struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wink"`
		Blush struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blush"`
		Innocent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"innocent"`
		SmilingFaceWith3Hearts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smiling_face_with_3_hearts"`
		HeartEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heart_eyes"`
		StarStruck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"star-struck"`
		KissingHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kissing_heart"`
		Kissing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kissing"`
		Relaxed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"relaxed"`
		KissingClosedEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kissing_closed_eyes"`
		KissingSmilingEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kissing_smiling_eyes"`
		SmilingFaceWithTear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smiling_face_with_tear"`
		Yum struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yum"`
		StuckOutTongue struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stuck_out_tongue"`
		StuckOutTongueWinkingEye struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stuck_out_tongue_winking_eye"`
		ZanyFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zany_face"`
		StuckOutTongueClosedEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stuck_out_tongue_closed_eyes"`
		MoneyMouthFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"money_mouth_face"`
		HuggingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hugging_face"`
		FaceWithHandOverMouth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_hand_over_mouth"`
		FaceWithOpenEyesAndHandOverMouth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_open_eyes_and_hand_over_mouth"`
		FaceWithPeekingEye struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_peeking_eye"`
		ShushingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shushing_face"`
		ThinkingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thinking_face"`
		SalutingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"saluting_face"`
		ZipperMouthFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zipper_mouth_face"`
		FaceWithRaisedEyebrow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_raised_eyebrow"`
		NeutralFace struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"neutral_face"`
		Expressionless struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"expressionless"`
		NoMouth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_mouth"`
		DottedLineFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dotted_line_face"`
		FaceInClouds struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"face_in_clouds"`
		Smirk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smirk"`
		Unamused struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"unamused"`
		FaceWithRollingEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_rolling_eyes"`
		Grimacing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grimacing"`
		FaceExhaling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"face_exhaling"`
		LyingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lying_face"`
		Relieved struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"relieved"`
		Pensive struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pensive"`
		Sleepy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sleepy"`
		DroolingFace struct {
			Id       string        `json:"id"`
			Name     string        `json:"name"`
			Keywords []interface{} `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"drooling_face"`
		Sleeping struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sleeping"`
		Mask struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mask"`
		FaceWithThermometer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_thermometer"`
		FaceWithHeadBandage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_head_bandage"`
		NauseatedFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nauseated_face"`
		FaceVomiting struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_vomiting"`
		SneezingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sneezing_face"`
		HotFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hot_face"`
		ColdFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cold_face"`
		WoozyFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woozy_face"`
		DizzyFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dizzy_face"`
		FaceWithSpiralEyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"face_with_spiral_eyes"`
		ExplodingHead struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"exploding_head"`
		FaceWithCowboyHat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_cowboy_hat"`
		PartyingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"partying_face"`
		DisguisedFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"disguised_face"`
		Sunglasses struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sunglasses"`
		NerdFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nerd_face"`
		FaceWithMonocle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_monocle"`
		Confused struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"confused"`
		FaceWithDiagonalMouth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_diagonal_mouth"`
		Worried struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"worried"`
		SlightlyFrowningFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"slightly_frowning_face"`
		WhiteFrowningFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_frowning_face"`
		OpenMouth struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"open_mouth"`
		Hushed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hushed"`
		Astonished struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"astonished"`
		Flushed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flushed"`
		PleadingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pleading_face"`
		FaceHoldingBackTears struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_holding_back_tears"`
		Frowning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"frowning"`
		Anguished struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"anguished"`
		Fearful struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fearful"`
		ColdSweat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cold_sweat"`
		DisappointedRelieved struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"disappointed_relieved"`
		Cry struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cry"`
		Sob struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sob"`
		Scream struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scream"`
		Confounded struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"confounded"`
		Persevere struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"persevere"`
		Disappointed struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"disappointed"`
		Sweat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sweat"`
		Weary struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"weary"`
		TiredFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tired_face"`
		YawningFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yawning_face"`
		Triumph struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"triumph"`
		Rage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rage"`
		Angry struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"angry"`
		FaceWithSymbolsOnMouth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_with_symbols_on_mouth"`
		SmilingImp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smiling_imp"`
		Imp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"imp"`
		Skull struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"skull"`
		SkullAndCrossbones struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"skull_and_crossbones"`
		Hankey struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hankey"`
		ClownFace struct {
			Id       string        `json:"id"`
			Name     string        `json:"name"`
			Keywords []interface{} `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clown_face"`
		JapaneseOgre struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"japanese_ogre"`
		JapaneseGoblin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"japanese_goblin"`
		Ghost struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ghost"`
		Alien struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"alien"`
		SpaceInvader struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"space_invader"`
		RobotFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"robot_face"`
		SmileyCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smiley_cat"`
		SmileCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smile_cat"`
		JoyCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"joy_cat"`
		HeartEyesCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heart_eyes_cat"`
		SmirkCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smirk_cat"`
		KissingCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kissing_cat"`
		ScreamCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scream_cat"`
		CryingCatFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crying_cat_face"`
		PoutingCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pouting_cat"`
		SeeNoEvil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"see_no_evil"`
		HearNoEvil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hear_no_evil"`
		SpeakNoEvil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"speak_no_evil"`
		Kiss struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kiss"`
		LoveLetter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"love_letter"`
		Cupid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cupid"`
		GiftHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gift_heart"`
		SparklingHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sparkling_heart"`
		Heartpulse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heartpulse"`
		Heartbeat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heartbeat"`
		RevolvingHearts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"revolving_hearts"`
		TwoHearts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"two_hearts"`
		HeartDecoration struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heart_decoration"`
		HeavyHeartExclamationMarkOrnament struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_heart_exclamation_mark_ornament"`
		BrokenHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"broken_heart"`
		HeartOnFire struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"heart_on_fire"`
		MendingHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"mending_heart"`
		Heart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heart"`
		OrangeHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"orange_heart"`
		YellowHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yellow_heart"`
		GreenHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"green_heart"`
		BlueHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blue_heart"`
		PurpleHeart struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"purple_heart"`
		BrownHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"brown_heart"`
		BlackHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_heart"`
		WhiteHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_heart"`
		Anger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"anger"`
		Boom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boom"`
		Dizzy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dizzy"`
		SweatDrops struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sweat_drops"`
		Dash struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dash"`
		Hole struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hole"`
		Bomb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bomb"`
		SpeechBalloon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"speech_balloon"`
		EyeInSpeechBubble struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eye-in-speech-bubble"`
		LeftSpeechBubble struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"left_speech_bubble"`
		RightAngerBubble struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"right_anger_bubble"`
		ThoughtBalloon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thought_balloon"`
		Zzz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zzz"`
		Wave struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wave"`
		RaisedBackOfHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"raised_back_of_hand"`
		RaisedHandWithFingersSplayed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"raised_hand_with_fingers_splayed"`
		Hand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hand"`
		SpockHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spock-hand"`
		RightwardsHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rightwards_hand"`
		LeftwardsHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leftwards_hand"`
		PalmDownHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"palm_down_hand"`
		PalmUpHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"palm_up_hand"`
		OkHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ok_hand"`
		PinchedFingers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pinched_fingers"`
		PinchingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pinching_hand"`
		V struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"v"`
		CrossedFingers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crossed_fingers"`
		HandWithIndexFingerAndThumbCrossed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hand_with_index_finger_and_thumb_crossed"`
		ILoveYouHandSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"i_love_you_hand_sign"`
		TheHorns struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"the_horns"`
		CallMeHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"call_me_hand"`
		PointLeft struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"point_left"`
		PointRight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"point_right"`
		PointUp2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"point_up_2"`
		MiddleFinger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"middle_finger"`
		PointDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"point_down"`
		PointUp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"point_up"`
		IndexPointingAtTheViewer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"index_pointing_at_the_viewer"`
		Field190 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"+1"`
		Field191 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"-1"`
		Fist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fist"`
		Facepunch struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"facepunch"`
		LeftFacingFist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"left-facing_fist"`
		RightFacingFist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"right-facing_fist"`
		Clap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clap"`
		RaisedHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"raised_hands"`
		HeartHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heart_hands"`
		OpenHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"open_hands"`
		PalmsUpTogether struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"palms_up_together"`
		Handshake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"handshake"`
		Pray struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pray"`
		WritingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"writing_hand"`
		NailCare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nail_care"`
		Selfie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"selfie"`
		Muscle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"muscle"`
		MechanicalArm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mechanical_arm"`
		MechanicalLeg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mechanical_leg"`
		Leg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leg"`
		Foot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"foot"`
		Ear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ear"`
		EarWithHearingAid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ear_with_hearing_aid"`
		Nose struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nose"`
		Brain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"brain"`
		AnatomicalHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"anatomical_heart"`
		Lungs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lungs"`
		Tooth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tooth"`
		Bone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bone"`
		Eyes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eyes"`
		Eye struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eye"`
		Tongue struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tongue"`
		Lips struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lips"`
		BitingLip struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"biting_lip"`
		Baby struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baby"`
		Child struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"child"`
		Boy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boy"`
		Girl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"girl"`
		Adult struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"adult"`
		PersonWithBlondHair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_with_blond_hair"`
		Man struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man"`
		BeardedPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bearded_person"`
		ManWithBeard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"man_with_beard"`
		WomanWithBeard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"woman_with_beard"`
		RedHairedMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"red_haired_man"`
		CurlyHairedMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"curly_haired_man"`
		WhiteHairedMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_haired_man"`
		BaldMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bald_man"`
		Woman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman"`
		RedHairedWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"red_haired_woman"`
		RedHairedPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"red_haired_person"`
		CurlyHairedWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"curly_haired_woman"`
		CurlyHairedPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"curly_haired_person"`
		WhiteHairedWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_haired_woman"`
		WhiteHairedPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"white_haired_person"`
		BaldWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bald_woman"`
		BaldPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"bald_person"`
		BlondHairedWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blond-haired-woman"`
		BlondHairedMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blond-haired-man"`
		OlderAdult struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"older_adult"`
		OlderMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"older_man"`
		OlderWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"older_woman"`
		PersonFrowning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_frowning"`
		ManFrowning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-frowning"`
		WomanFrowning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-frowning"`
		PersonWithPoutingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_with_pouting_face"`
		ManPouting struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-pouting"`
		WomanPouting struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-pouting"`
		NoGood struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_good"`
		ManGesturingNo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-gesturing-no"`
		WomanGesturingNo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-gesturing-no"`
		OkWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ok_woman"`
		ManGesturingOk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-gesturing-ok"`
		WomanGesturingOk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-gesturing-ok"`
		InformationDeskPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"information_desk_person"`
		ManTippingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-tipping-hand"`
		WomanTippingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-tipping-hand"`
		RaisingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"raising_hand"`
		ManRaisingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-raising-hand"`
		WomanRaisingHand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-raising-hand"`
		DeafPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"deaf_person"`
		DeafMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"deaf_man"`
		DeafWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"deaf_woman"`
		Bow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bow"`
		ManBowing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-bowing"`
		WomanBowing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-bowing"`
		FacePalm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"face_palm"`
		ManFacepalming struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-facepalming"`
		WomanFacepalming struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-facepalming"`
		Shrug struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shrug"`
		ManShrugging struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-shrugging"`
		WomanShrugging struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-shrugging"`
		HealthWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"health_worker"`
		MaleDoctor struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-doctor"`
		FemaleDoctor struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-doctor"`
		Student struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"student"`
		MaleStudent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-student"`
		FemaleStudent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-student"`
		Teacher struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"teacher"`
		MaleTeacher struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-teacher"`
		FemaleTeacher struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-teacher"`
		Judge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"judge"`
		MaleJudge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-judge"`
		FemaleJudge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-judge"`
		Farmer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"farmer"`
		MaleFarmer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-farmer"`
		FemaleFarmer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-farmer"`
		Cook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"cook"`
		MaleCook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-cook"`
		FemaleCook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-cook"`
		Mechanic struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"mechanic"`
		MaleMechanic struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-mechanic"`
		FemaleMechanic struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-mechanic"`
		FactoryWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"factory_worker"`
		MaleFactoryWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-factory-worker"`
		FemaleFactoryWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-factory-worker"`
		OfficeWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"office_worker"`
		MaleOfficeWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-office-worker"`
		FemaleOfficeWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-office-worker"`
		Scientist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"scientist"`
		MaleScientist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-scientist"`
		FemaleScientist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-scientist"`
		Technologist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"technologist"`
		MaleTechnologist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-technologist"`
		FemaleTechnologist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-technologist"`
		Singer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"singer"`
		MaleSinger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-singer"`
		FemaleSinger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-singer"`
		Artist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"artist"`
		MaleArtist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-artist"`
		FemaleArtist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-artist"`
		Pilot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"pilot"`
		MalePilot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-pilot"`
		FemalePilot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-pilot"`
		Astronaut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"astronaut"`
		MaleAstronaut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-astronaut"`
		FemaleAstronaut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-astronaut"`
		Firefighter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"firefighter"`
		MaleFirefighter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-firefighter"`
		FemaleFirefighter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-firefighter"`
		Cop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cop"`
		MalePoliceOfficer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-police-officer"`
		FemalePoliceOfficer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-police-officer"`
		SleuthOrSpy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sleuth_or_spy"`
		MaleDetective struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-detective"`
		FemaleDetective struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-detective"`
		Guardsman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"guardsman"`
		MaleGuard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-guard"`
		FemaleGuard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-guard"`
		Ninja struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ninja"`
		ConstructionWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"construction_worker"`
		MaleConstructionWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male-construction-worker"`
		FemaleConstructionWorker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female-construction-worker"`
		PersonWithCrown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_with_crown"`
		Prince struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"prince"`
		Princess struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"princess"`
		ManWithTurban struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_with_turban"`
		ManWearingTurban struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-wearing-turban"`
		WomanWearingTurban struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-wearing-turban"`
		ManWithGuaPiMao struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_with_gua_pi_mao"`
		PersonWithHeadscarf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_with_headscarf"`
		PersonInTuxedo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_in_tuxedo"`
		ManInTuxedo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_tuxedo"`
		WomanInTuxedo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_in_tuxedo"`
		BrideWithVeil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bride_with_veil"`
		ManWithVeil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_with_veil"`
		WomanWithVeil struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_with_veil"`
		PregnantWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pregnant_woman"`
		PregnantMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pregnant_man"`
		PregnantPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pregnant_person"`
		BreastFeeding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"breast-feeding"`
		WomanFeedingBaby struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_feeding_baby"`
		ManFeedingBaby struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_feeding_baby"`
		PersonFeedingBaby struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_feeding_baby"`
		Angel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"angel"`
		Santa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"santa"`
		MrsClaus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mrs_claus"`
		MxClaus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mx_claus"`
		Superhero struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"superhero"`
		MaleSuperhero struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_superhero"`
		FemaleSuperhero struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_superhero"`
		Supervillain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"supervillain"`
		MaleSupervillain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_supervillain"`
		FemaleSupervillain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_supervillain"`
		Mage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mage"`
		MaleMage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_mage"`
		FemaleMage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_mage"`
		Fairy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fairy"`
		MaleFairy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_fairy"`
		FemaleFairy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_fairy"`
		Vampire struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"vampire"`
		MaleVampire struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_vampire"`
		FemaleVampire struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_vampire"`
		Merperson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"merperson"`
		Merman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"merman"`
		Mermaid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mermaid"`
		Elf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"elf"`
		MaleElf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_elf"`
		FemaleElf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_elf"`
		Genie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"genie"`
		MaleGenie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_genie"`
		FemaleGenie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_genie"`
		Zombie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zombie"`
		MaleZombie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_zombie"`
		FemaleZombie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_zombie"`
		Troll struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"troll"`
		Massage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"massage"`
		ManGettingMassage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-getting-massage"`
		WomanGettingMassage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-getting-massage"`
		Haircut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"haircut"`
		ManGettingHaircut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-getting-haircut"`
		WomanGettingHaircut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-getting-haircut"`
		Walking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"walking"`
		ManWalking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-walking"`
		WomanWalking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-walking"`
		StandingPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"standing_person"`
		ManStanding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_standing"`
		WomanStanding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_standing"`
		KneelingPerson struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kneeling_person"`
		ManKneeling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_kneeling"`
		WomanKneeling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_kneeling"`
		PersonWithProbingCane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"person_with_probing_cane"`
		ManWithProbingCane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_with_probing_cane"`
		WomanWithProbingCane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_with_probing_cane"`
		PersonInMotorizedWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"person_in_motorized_wheelchair"`
		ManInMotorizedWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_motorized_wheelchair"`
		WomanInMotorizedWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_in_motorized_wheelchair"`
		PersonInManualWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version float64 `json:"version"`
		} `json:"person_in_manual_wheelchair"`
		ManInManualWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_manual_wheelchair"`
		WomanInManualWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_in_manual_wheelchair"`
		Runner struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"runner"`
		ManRunning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-running"`
		WomanRunning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-running"`
		Dancer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dancer"`
		ManDancing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_dancing"`
		ManInBusinessSuitLevitating struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_business_suit_levitating"`
		Dancers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dancers"`
		MenWithBunnyEarsPartying struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"men-with-bunny-ears-partying"`
		WomenWithBunnyEarsPartying struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"women-with-bunny-ears-partying"`
		PersonInSteamyRoom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_in_steamy_room"`
		ManInSteamyRoom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_steamy_room"`
		WomanInSteamyRoom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_in_steamy_room"`
		PersonClimbing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_climbing"`
		ManClimbing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_climbing"`
		WomanClimbing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_climbing"`
		Fencer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fencer"`
		HorseRacing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"horse_racing"`
		Skier struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"skier"`
		Snowboarder struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snowboarder"`
		Golfer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"golfer"`
		ManGolfing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-golfing"`
		WomanGolfing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-golfing"`
		Surfer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"surfer"`
		ManSurfing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-surfing"`
		WomanSurfing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-surfing"`
		Rowboat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rowboat"`
		ManRowingBoat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-rowing-boat"`
		WomanRowingBoat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-rowing-boat"`
		Swimmer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"swimmer"`
		ManSwimming struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-swimming"`
		WomanSwimming struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-swimming"`
		PersonWithBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_with_ball"`
		ManBouncingBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-bouncing-ball"`
		WomanBouncingBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-bouncing-ball"`
		WeightLifter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"weight_lifter"`
		ManLiftingWeights struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-lifting-weights"`
		WomanLiftingWeights struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-lifting-weights"`
		Bicyclist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bicyclist"`
		ManBiking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-biking"`
		WomanBiking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-biking"`
		MountainBicyclist struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mountain_bicyclist"`
		ManMountainBiking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-mountain-biking"`
		WomanMountainBiking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-mountain-biking"`
		PersonDoingCartwheel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_doing_cartwheel"`
		ManCartwheeling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-cartwheeling"`
		WomanCartwheeling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-cartwheeling"`
		Wrestlers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wrestlers"`
		ManWrestling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-wrestling"`
		WomanWrestling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-wrestling"`
		WaterPolo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"water_polo"`
		ManPlayingWaterPolo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-playing-water-polo"`
		WomanPlayingWaterPolo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-playing-water-polo"`
		Handball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"handball"`
		ManPlayingHandball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-playing-handball"`
		WomanPlayingHandball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-playing-handball"`
		Juggling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"juggling"`
		ManJuggling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-juggling"`
		WomanJuggling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-juggling"`
		PersonInLotusPosition struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"person_in_lotus_position"`
		ManInLotusPosition struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_in_lotus_position"`
		WomanInLotusPosition struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman_in_lotus_position"`
		Bath struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bath"`
		SleepingAccommodation struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sleeping_accommodation"`
		PeopleHoldingHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"people_holding_hands"`
		TwoWomenHoldingHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"two_women_holding_hands"`
		ManAndWomanHoldingHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man_and_woman_holding_hands"`
		TwoMenHoldingHands struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"two_men_holding_hands"`
		Couplekiss struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"couplekiss"`
		WomanKissMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-kiss-man"`
		ManKissMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-kiss-man"`
		WomanKissWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-kiss-woman"`
		CoupleWithHeart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"couple_with_heart"`
		WomanHeartMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-heart-man"`
		ManHeartMan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-heart-man"`
		WomanHeartWoman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-heart-woman"`
		Family struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"family"`
		ManWomanBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-woman-boy"`
		ManWomanGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-woman-girl"`
		ManWomanGirlBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-woman-girl-boy"`
		ManWomanBoyBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-woman-boy-boy"`
		ManWomanGirlGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-woman-girl-girl"`
		ManManBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-man-boy"`
		ManManGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-man-girl"`
		ManManGirlBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-man-girl-boy"`
		ManManBoyBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-man-boy-boy"`
		ManManGirlGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-man-girl-girl"`
		WomanWomanBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-woman-boy"`
		WomanWomanGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-woman-girl"`
		WomanWomanGirlBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-woman-girl-boy"`
		WomanWomanBoyBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-woman-boy-boy"`
		WomanWomanGirlGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-woman-girl-girl"`
		ManBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-boy"`
		ManBoyBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-boy-boy"`
		ManGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-girl"`
		ManGirlBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-girl-boy"`
		ManGirlGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"man-girl-girl"`
		WomanBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-boy"`
		WomanBoyBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-boy-boy"`
		WomanGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-girl"`
		WomanGirlBoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-girl-boy"`
		WomanGirlGirl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"woman-girl-girl"`
		SpeakingHeadInSilhouette struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"speaking_head_in_silhouette"`
		BustInSilhouette struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bust_in_silhouette"`
		BustsInSilhouette struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"busts_in_silhouette"`
		PeopleHugging struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"people_hugging"`
		Footprints struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"footprints"`
		MonkeyFace struct {
			Id        string   `json:"id"`
			Name      string   `json:"name"`
			Emoticons []string `json:"emoticons"`
			Keywords  []string `json:"keywords"`
			Skins     []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"monkey_face"`
		Monkey struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"monkey"`
		Gorilla struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gorilla"`
		Orangutan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"orangutan"`
		Dog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dog"`
		Dog2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dog2"`
		GuideDog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"guide_dog"`
		ServiceDog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"service_dog"`
		Poodle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"poodle"`
		Wolf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wolf"`
		FoxFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fox_face"`
		Raccoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"raccoon"`
		Cat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cat"`
		Cat2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cat2"`
		BlackCat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_cat"`
		LionFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lion_face"`
		Tiger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tiger"`
		Tiger2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tiger2"`
		Leopard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leopard"`
		Horse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"horse"`
		Racehorse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"racehorse"`
		UnicornFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"unicorn_face"`
		ZebraFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zebra_face"`
		Deer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"deer"`
		Bison struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bison"`
		Cow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cow"`
		Ox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ox"`
		WaterBuffalo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"water_buffalo"`
		Cow2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cow2"`
		Pig struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pig"`
		Pig2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pig2"`
		Boar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boar"`
		PigNose struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pig_nose"`
		Ram struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ram"`
		Sheep struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sheep"`
		Goat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"goat"`
		DromedaryCamel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dromedary_camel"`
		Camel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"camel"`
		Llama struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"llama"`
		GiraffeFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"giraffe_face"`
		Elephant struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"elephant"`
		Mammoth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mammoth"`
		Rhinoceros struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rhinoceros"`
		Hippopotamus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hippopotamus"`
		Mouse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mouse"`
		Mouse2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mouse2"`
		Rat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rat"`
		Hamster struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hamster"`
		Rabbit struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rabbit"`
		Rabbit2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rabbit2"`
		Chipmunk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chipmunk"`
		Beaver struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beaver"`
		Hedgehog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hedgehog"`
		Bat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bat"`
		Bear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bear"`
		PolarBear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"polar_bear"`
		Koala struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"koala"`
		PandaFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"panda_face"`
		Sloth struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sloth"`
		Otter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"otter"`
		Skunk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"skunk"`
		Kangaroo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kangaroo"`
		Badger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"badger"`
		Feet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"feet"`
		Turkey struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"turkey"`
		Chicken struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chicken"`
		Rooster struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rooster"`
		HatchingChick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hatching_chick"`
		BabyChick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baby_chick"`
		HatchedChick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hatched_chick"`
		Bird struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bird"`
		Penguin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"penguin"`
		DoveOfPeace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dove_of_peace"`
		Eagle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eagle"`
		Duck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"duck"`
		Swan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"swan"`
		Owl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"owl"`
		Dodo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dodo"`
		Feather struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"feather"`
		Flamingo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flamingo"`
		Peacock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"peacock"`
		Parrot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"parrot"`
		Frog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"frog"`
		Crocodile struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crocodile"`
		Turtle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"turtle"`
		Lizard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lizard"`
		Snake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snake"`
		DragonFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dragon_face"`
		Dragon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dragon"`
		Sauropod struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sauropod"`
		TRex struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"t-rex"`
		Whale struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"whale"`
		Whale2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"whale2"`
		Dolphin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dolphin"`
		Seal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"seal"`
		Fish struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fish"`
		TropicalFish struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tropical_fish"`
		Blowfish struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blowfish"`
		Shark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shark"`
		Octopus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"octopus"`
		Shell struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shell"`
		Coral struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coral"`
		Snail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snail"`
		Butterfly struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"butterfly"`
		Bug struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bug"`
		Ant struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ant"`
		Bee struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bee"`
		Beetle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beetle"`
		Ladybug struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ladybug"`
		Cricket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cricket"`
		Cockroach struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cockroach"`
		Spider struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spider"`
		SpiderWeb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spider_web"`
		Scorpion struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scorpion"`
		Mosquito struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mosquito"`
		Fly struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fly"`
		Worm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"worm"`
		Microbe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"microbe"`
		Bouquet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bouquet"`
		CherryBlossom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cherry_blossom"`
		WhiteFlower struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_flower"`
		Lotus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lotus"`
		Rosette struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rosette"`
		Rose struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rose"`
		WiltedFlower struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wilted_flower"`
		Hibiscus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hibiscus"`
		Sunflower struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sunflower"`
		Blossom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blossom"`
		Tulip struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tulip"`
		Seedling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"seedling"`
		PottedPlant struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"potted_plant"`
		EvergreenTree struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"evergreen_tree"`
		DeciduousTree struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"deciduous_tree"`
		PalmTree struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"palm_tree"`
		Cactus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cactus"`
		EarOfRice struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ear_of_rice"`
		Herb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"herb"`
		Shamrock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shamrock"`
		FourLeafClover struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"four_leaf_clover"`
		MapleLeaf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"maple_leaf"`
		FallenLeaf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fallen_leaf"`
		Leaves struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leaves"`
		EmptyNest struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"empty_nest"`
		NestWithEggs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nest_with_eggs"`
		Grapes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grapes"`
		Melon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"melon"`
		Watermelon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"watermelon"`
		Tangerine struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tangerine"`
		Lemon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lemon"`
		Banana struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"banana"`
		Pineapple struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pineapple"`
		Mango struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mango"`
		Apple struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"apple"`
		GreenApple struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"green_apple"`
		Pear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pear"`
		Peach struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"peach"`
		Cherries struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cherries"`
		Strawberry struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"strawberry"`
		Blueberries struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blueberries"`
		Kiwifruit struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kiwifruit"`
		Tomato struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tomato"`
		Olive struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"olive"`
		Coconut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coconut"`
		Avocado struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"avocado"`
		Eggplant struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eggplant"`
		Potato struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"potato"`
		Carrot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"carrot"`
		Corn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"corn"`
		HotPepper struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hot_pepper"`
		BellPepper struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bell_pepper"`
		Cucumber struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cucumber"`
		LeafyGreen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leafy_green"`
		Broccoli struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"broccoli"`
		Garlic struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"garlic"`
		Onion struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"onion"`
		Mushroom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mushroom"`
		Peanuts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"peanuts"`
		Beans struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beans"`
		Chestnut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chestnut"`
		Bread struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bread"`
		Croissant struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"croissant"`
		BaguetteBread struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baguette_bread"`
		Flatbread struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flatbread"`
		Pretzel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pretzel"`
		Bagel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bagel"`
		Pancakes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pancakes"`
		Waffle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waffle"`
		CheeseWedge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cheese_wedge"`
		MeatOnBone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"meat_on_bone"`
		PoultryLeg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"poultry_leg"`
		CutOfMeat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cut_of_meat"`
		Bacon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bacon"`
		Hamburger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hamburger"`
		Fries struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fries"`
		Pizza struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pizza"`
		Hotdog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hotdog"`
		Sandwich struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sandwich"`
		Taco struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"taco"`
		Burrito struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"burrito"`
		Tamale struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tamale"`
		StuffedFlatbread struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stuffed_flatbread"`
		Falafel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"falafel"`
		Egg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"egg"`
		FriedEgg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fried_egg"`
		ShallowPanOfFood struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shallow_pan_of_food"`
		Stew struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stew"`
		Fondue struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fondue"`
		BowlWithSpoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bowl_with_spoon"`
		GreenSalad struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"green_salad"`
		Popcorn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"popcorn"`
		Butter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"butter"`
		Salt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"salt"`
		CannedFood struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"canned_food"`
		Bento struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bento"`
		RiceCracker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rice_cracker"`
		RiceBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rice_ball"`
		Rice struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rice"`
		Curry struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"curry"`
		Ramen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ramen"`
		Spaghetti struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spaghetti"`
		SweetPotato struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sweet_potato"`
		Oden struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oden"`
		Sushi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sushi"`
		FriedShrimp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fried_shrimp"`
		FishCake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fish_cake"`
		MoonCake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"moon_cake"`
		Dango struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dango"`
		Dumpling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dumpling"`
		FortuneCookie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fortune_cookie"`
		TakeoutBox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"takeout_box"`
		Crab struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crab"`
		Lobster struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lobster"`
		Shrimp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shrimp"`
		Squid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"squid"`
		Oyster struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oyster"`
		Icecream struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"icecream"`
		ShavedIce struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shaved_ice"`
		IceCream struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ice_cream"`
		Doughnut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"doughnut"`
		Cookie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cookie"`
		Birthday struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"birthday"`
		Cake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cake"`
		Cupcake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cupcake"`
		Pie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pie"`
		ChocolateBar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chocolate_bar"`
		Candy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"candy"`
		Lollipop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lollipop"`
		Custard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"custard"`
		HoneyPot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"honey_pot"`
		BabyBottle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baby_bottle"`
		GlassOfMilk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"glass_of_milk"`
		Coffee struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coffee"`
		Teapot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"teapot"`
		Tea struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tea"`
		Sake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sake"`
		Champagne struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"champagne"`
		WineGlass struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wine_glass"`
		Cocktail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cocktail"`
		TropicalDrink struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tropical_drink"`
		Beer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beer"`
		Beers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beers"`
		ClinkingGlasses struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clinking_glasses"`
		TumblerGlass struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tumbler_glass"`
		PouringLiquid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pouring_liquid"`
		CupWithStraw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cup_with_straw"`
		BubbleTea struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bubble_tea"`
		BeverageBox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beverage_box"`
		MateDrink struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mate_drink"`
		IceCube struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ice_cube"`
		Chopsticks struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chopsticks"`
		KnifeForkPlate struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"knife_fork_plate"`
		ForkAndKnife struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fork_and_knife"`
		Spoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spoon"`
		Hocho struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hocho"`
		Jar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"jar"`
		Amphora struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"amphora"`
		EarthAfrica struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"earth_africa"`
		EarthAmericas struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"earth_americas"`
		EarthAsia struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"earth_asia"`
		GlobeWithMeridians struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"globe_with_meridians"`
		WorldMap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"world_map"`
		Japan struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"japan"`
		Compass struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"compass"`
		SnowCappedMountain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snow_capped_mountain"`
		Mountain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mountain"`
		Volcano struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"volcano"`
		MountFuji struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mount_fuji"`
		Camping struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"camping"`
		BeachWithUmbrella struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beach_with_umbrella"`
		Desert struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"desert"`
		DesertIsland struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"desert_island"`
		NationalPark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"national_park"`
		Stadium struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stadium"`
		ClassicalBuilding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"classical_building"`
		BuildingConstruction struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"building_construction"`
		Bricks struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bricks"`
		Rock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rock"`
		Wood struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wood"`
		Hut struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hut"`
		HouseBuildings struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"house_buildings"`
		DerelictHouseBuilding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"derelict_house_building"`
		House struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"house"`
		HouseWithGarden struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"house_with_garden"`
		Office struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"office"`
		PostOffice struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"post_office"`
		EuropeanPostOffice struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"european_post_office"`
		Hospital struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hospital"`
		Bank struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bank"`
		Hotel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hotel"`
		LoveHotel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"love_hotel"`
		ConvenienceStore struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"convenience_store"`
		School struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"school"`
		DepartmentStore struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"department_store"`
		Factory struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"factory"`
		JapaneseCastle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"japanese_castle"`
		EuropeanCastle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"european_castle"`
		Wedding struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wedding"`
		TokyoTower struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tokyo_tower"`
		StatueOfLiberty struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"statue_of_liberty"`
		Church struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"church"`
		Mosque struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mosque"`
		HinduTemple struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hindu_temple"`
		Synagogue struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"synagogue"`
		ShintoShrine struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shinto_shrine"`
		Kaaba struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kaaba"`
		Fountain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fountain"`
		Tent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tent"`
		Foggy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"foggy"`
		NightWithStars struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"night_with_stars"`
		Cityscape struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cityscape"`
		SunriseOverMountains struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sunrise_over_mountains"`
		Sunrise struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sunrise"`
		CitySunset struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"city_sunset"`
		CitySunrise struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"city_sunrise"`
		BridgeAtNight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bridge_at_night"`
		Hotsprings struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hotsprings"`
		CarouselHorse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"carousel_horse"`
		PlaygroundSlide struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"playground_slide"`
		FerrisWheel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ferris_wheel"`
		RollerCoaster struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"roller_coaster"`
		Barber struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"barber"`
		CircusTent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"circus_tent"`
		SteamLocomotive struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"steam_locomotive"`
		RailwayCar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"railway_car"`
		BullettrainSide struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bullettrain_side"`
		BullettrainFront struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bullettrain_front"`
		Train2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"train2"`
		Metro struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"metro"`
		LightRail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"light_rail"`
		Station struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"station"`
		Tram struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tram"`
		Monorail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"monorail"`
		MountainRailway struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mountain_railway"`
		Train struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"train"`
		Bus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bus"`
		OncomingBus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oncoming_bus"`
		Trolleybus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"trolleybus"`
		Minibus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"minibus"`
		Ambulance struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ambulance"`
		FireEngine struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fire_engine"`
		PoliceCar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"police_car"`
		OncomingPoliceCar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oncoming_police_car"`
		Taxi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"taxi"`
		OncomingTaxi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oncoming_taxi"`
		Car struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"car"`
		OncomingAutomobile struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oncoming_automobile"`
		BlueCar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blue_car"`
		PickupTruck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pickup_truck"`
		Truck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"truck"`
		ArticulatedLorry struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"articulated_lorry"`
		Tractor struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tractor"`
		RacingCar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"racing_car"`
		RacingMotorcycle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"racing_motorcycle"`
		MotorScooter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"motor_scooter"`
		ManualWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"manual_wheelchair"`
		MotorizedWheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"motorized_wheelchair"`
		AutoRickshaw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"auto_rickshaw"`
		Bike struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bike"`
		Scooter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scooter"`
		Skateboard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"skateboard"`
		RollerSkate struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"roller_skate"`
		Busstop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"busstop"`
		Motorway struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"motorway"`
		RailwayTrack struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"railway_track"`
		OilDrum struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"oil_drum"`
		Fuelpump struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fuelpump"`
		Wheel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wheel"`
		RotatingLight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rotating_light"`
		TrafficLight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"traffic_light"`
		VerticalTrafficLight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"vertical_traffic_light"`
		OctagonalSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"octagonal_sign"`
		Construction struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"construction"`
		Anchor struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"anchor"`
		RingBuoy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ring_buoy"`
		Boat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boat"`
		Canoe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"canoe"`
		Speedboat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"speedboat"`
		PassengerShip struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"passenger_ship"`
		Ferry struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ferry"`
		MotorBoat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"motor_boat"`
		Ship struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ship"`
		Airplane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"airplane"`
		SmallAirplane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"small_airplane"`
		AirplaneDeparture struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"airplane_departure"`
		AirplaneArriving struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"airplane_arriving"`
		Parachute struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"parachute"`
		Seat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"seat"`
		Helicopter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"helicopter"`
		SuspensionRailway struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"suspension_railway"`
		MountainCableway struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mountain_cableway"`
		AerialTramway struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"aerial_tramway"`
		Satellite struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"satellite"`
		Rocket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rocket"`
		FlyingSaucer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flying_saucer"`
		BellhopBell struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bellhop_bell"`
		Luggage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"luggage"`
		Hourglass struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hourglass"`
		HourglassFlowingSand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hourglass_flowing_sand"`
		Watch struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"watch"`
		AlarmClock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"alarm_clock"`
		Stopwatch struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stopwatch"`
		TimerClock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"timer_clock"`
		MantelpieceClock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mantelpiece_clock"`
		Clock12 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock12"`
		Clock1230 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock1230"`
		Clock1 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock1"`
		Clock130 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock130"`
		Clock2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock2"`
		Clock230 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock230"`
		Clock3 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock3"`
		Clock330 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock330"`
		Clock4 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock4"`
		Clock430 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock430"`
		Clock5 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock5"`
		Clock530 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock530"`
		Clock6 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock6"`
		Clock630 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock630"`
		Clock7 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock7"`
		Clock730 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock730"`
		Clock8 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock8"`
		Clock830 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock830"`
		Clock9 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock9"`
		Clock930 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock930"`
		Clock10 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock10"`
		Clock1030 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock1030"`
		Clock11 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock11"`
		Clock1130 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clock1130"`
		NewMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"new_moon"`
		WaxingCrescentMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waxing_crescent_moon"`
		FirstQuarterMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"first_quarter_moon"`
		Moon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"moon"`
		FullMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"full_moon"`
		WaningGibbousMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waning_gibbous_moon"`
		LastQuarterMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"last_quarter_moon"`
		WaningCrescentMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waning_crescent_moon"`
		CrescentMoon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crescent_moon"`
		NewMoonWithFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"new_moon_with_face"`
		FirstQuarterMoonWithFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"first_quarter_moon_with_face"`
		LastQuarterMoonWithFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"last_quarter_moon_with_face"`
		Thermometer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thermometer"`
		Sunny struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sunny"`
		FullMoonWithFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"full_moon_with_face"`
		SunWithFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sun_with_face"`
		RingedPlanet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ringed_planet"`
		Star struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"star"`
		Star2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"star2"`
		Stars struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stars"`
		MilkyWay struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"milky_way"`
		Cloud struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cloud"`
		PartlySunny struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"partly_sunny"`
		ThunderCloudAndRain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thunder_cloud_and_rain"`
		MostlySunny struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mostly_sunny"`
		BarelySunny struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"barely_sunny"`
		PartlySunnyRain struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"partly_sunny_rain"`
		RainCloud struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rain_cloud"`
		SnowCloud struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snow_cloud"`
		Lightning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lightning"`
		Tornado struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tornado"`
		Fog struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fog"`
		WindBlowingFace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wind_blowing_face"`
		Cyclone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cyclone"`
		Rainbow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rainbow"`
		ClosedUmbrella struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"closed_umbrella"`
		Umbrella struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"umbrella"`
		UmbrellaWithRainDrops struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"umbrella_with_rain_drops"`
		UmbrellaOnGround struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"umbrella_on_ground"`
		Zap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zap"`
		Snowflake struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snowflake"`
		Snowman struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snowman"`
		SnowmanWithoutSnow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"snowman_without_snow"`
		Comet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"comet"`
		Fire struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fire"`
		Droplet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"droplet"`
		Ocean struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ocean"`
		JackOLantern struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"jack_o_lantern"`
		ChristmasTree struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"christmas_tree"`
		Fireworks struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fireworks"`
		Sparkler struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sparkler"`
		Firecracker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"firecracker"`
		Sparkles struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sparkles"`
		Balloon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"balloon"`
		Tada struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tada"`
		ConfettiBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"confetti_ball"`
		TanabataTree struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tanabata_tree"`
		Bamboo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bamboo"`
		Dolls struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dolls"`
		Flags struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flags"`
		WindChime struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wind_chime"`
		RiceScene struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rice_scene"`
		RedEnvelope struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"red_envelope"`
		Ribbon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ribbon"`
		Gift struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gift"`
		ReminderRibbon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"reminder_ribbon"`
		AdmissionTickets struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"admission_tickets"`
		Ticket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ticket"`
		Medal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"medal"`
		Trophy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"trophy"`
		SportsMedal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sports_medal"`
		FirstPlaceMedal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"first_place_medal"`
		SecondPlaceMedal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"second_place_medal"`
		ThirdPlaceMedal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"third_place_medal"`
		Soccer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"soccer"`
		Baseball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baseball"`
		Softball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"softball"`
		Basketball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"basketball"`
		Volleyball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"volleyball"`
		Football struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"football"`
		RugbyFootball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rugby_football"`
		Tennis struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tennis"`
		FlyingDisc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flying_disc"`
		Bowling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bowling"`
		CricketBatAndBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cricket_bat_and_ball"`
		FieldHockeyStickAndBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"field_hockey_stick_and_ball"`
		IceHockeyStickAndPuck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ice_hockey_stick_and_puck"`
		Lacrosse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lacrosse"`
		TableTennisPaddleAndBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"table_tennis_paddle_and_ball"`
		BadmintonRacquetAndShuttlecock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"badminton_racquet_and_shuttlecock"`
		BoxingGlove struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boxing_glove"`
		MartialArtsUniform struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"martial_arts_uniform"`
		GoalNet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"goal_net"`
		Golf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"golf"`
		IceSkate struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ice_skate"`
		FishingPoleAndFish struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fishing_pole_and_fish"`
		DivingMask struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"diving_mask"`
		RunningShirtWithSash struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"running_shirt_with_sash"`
		Ski struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ski"`
		Sled struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sled"`
		CurlingStone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"curling_stone"`
		Dart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dart"`
		YoYo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yo-yo"`
		Kite struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kite"`
		Ball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"8ball"`
		CrystalBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crystal_ball"`
		MagicWand struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"magic_wand"`
		NazarAmulet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nazar_amulet"`
		Hamsa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hamsa"`
		VideoGame struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"video_game"`
		Joystick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"joystick"`
		SlotMachine struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"slot_machine"`
		GameDie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"game_die"`
		Jigsaw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"jigsaw"`
		TeddyBear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"teddy_bear"`
		Pinata struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pinata"`
		MirrorBall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mirror_ball"`
		NestingDolls struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nesting_dolls"`
		Spades struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spades"`
		Hearts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hearts"`
		Diamonds struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"diamonds"`
		Clubs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clubs"`
		ChessPawn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chess_pawn"`
		BlackJoker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_joker"`
		Mahjong struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mahjong"`
		FlowerPlayingCards struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flower_playing_cards"`
		PerformingArts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"performing_arts"`
		FrameWithPicture struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"frame_with_picture"`
		Art struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"art"`
		Thread struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thread"`
		SewingNeedle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sewing_needle"`
		Yarn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yarn"`
		Knot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"knot"`
		Eyeglasses struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eyeglasses"`
		DarkSunglasses struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dark_sunglasses"`
		Goggles struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"goggles"`
		LabCoat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lab_coat"`
		SafetyVest struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"safety_vest"`
		Necktie struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"necktie"`
		Shirt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shirt"`
		Jeans struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"jeans"`
		Scarf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scarf"`
		Gloves struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gloves"`
		Coat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coat"`
		Socks struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"socks"`
		Dress struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dress"`
		Kimono struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kimono"`
		Sari struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sari"`
		OnePieceSwimsuit struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"one-piece_swimsuit"`
		Briefs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"briefs"`
		Shorts struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shorts"`
		Bikini struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bikini"`
		WomansClothes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"womans_clothes"`
		Purse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"purse"`
		Handbag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"handbag"`
		Pouch struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pouch"`
		ShoppingBags struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shopping_bags"`
		SchoolSatchel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"school_satchel"`
		ThongSandal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"thong_sandal"`
		MansShoe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mans_shoe"`
		AthleticShoe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"athletic_shoe"`
		HikingBoot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hiking_boot"`
		WomansFlatShoe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"womans_flat_shoe"`
		HighHeel struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"high_heel"`
		Sandal struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sandal"`
		BalletShoes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ballet_shoes"`
		Boot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boot"`
		Crown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crown"`
		WomansHat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"womans_hat"`
		Tophat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tophat"`
		MortarBoard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mortar_board"`
		BilledCap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"billed_cap"`
		MilitaryHelmet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"military_helmet"`
		HelmetWithWhiteCross struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"helmet_with_white_cross"`
		PrayerBeads struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"prayer_beads"`
		Lipstick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lipstick"`
		Ring struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ring"`
		Gem struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gem"`
		Mute struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mute"`
		Speaker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"speaker"`
		Sound struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sound"`
		LoudSound struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"loud_sound"`
		Loudspeaker struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"loudspeaker"`
		Mega struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mega"`
		PostalHorn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"postal_horn"`
		Bell struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bell"`
		NoBell struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_bell"`
		MusicalScore struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"musical_score"`
		MusicalNote struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"musical_note"`
		Notes struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"notes"`
		StudioMicrophone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"studio_microphone"`
		LevelSlider struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"level_slider"`
		ControlKnobs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"control_knobs"`
		Microphone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"microphone"`
		Headphones struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"headphones"`
		Radio struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"radio"`
		Saxophone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"saxophone"`
		Accordion struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"accordion"`
		Guitar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"guitar"`
		MusicalKeyboard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"musical_keyboard"`
		Trumpet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"trumpet"`
		Violin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"violin"`
		Banjo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"banjo"`
		DrumWithDrumsticks struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"drum_with_drumsticks"`
		LongDrum struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"long_drum"`
		Iphone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"iphone"`
		Calling struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"calling"`
		Phone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"phone"`
		TelephoneReceiver struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"telephone_receiver"`
		Pager struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pager"`
		Fax struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fax"`
		Battery struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"battery"`
		LowBattery struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"low_battery"`
		ElectricPlug struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"electric_plug"`
		Computer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"computer"`
		DesktopComputer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"desktop_computer"`
		Printer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"printer"`
		Keyboard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"keyboard"`
		ThreeButtonMouse struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"three_button_mouse"`
		Trackball struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"trackball"`
		Minidisc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"minidisc"`
		FloppyDisk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"floppy_disk"`
		Cd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cd"`
		Dvd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dvd"`
		Abacus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"abacus"`
		MovieCamera struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"movie_camera"`
		FilmFrames struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"film_frames"`
		FilmProjector struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"film_projector"`
		Clapper struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clapper"`
		Tv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tv"`
		Camera struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"camera"`
		CameraWithFlash struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"camera_with_flash"`
		VideoCamera struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"video_camera"`
		Vhs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"vhs"`
		Mag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mag"`
		MagRight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mag_right"`
		Candle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"candle"`
		Bulb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bulb"`
		Flashlight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flashlight"`
		IzakayaLantern struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"izakaya_lantern"`
		DiyaLamp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"diya_lamp"`
		NotebookWithDecorativeCover struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"notebook_with_decorative_cover"`
		ClosedBook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"closed_book"`
		Book struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"book"`
		GreenBook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"green_book"`
		BlueBook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"blue_book"`
		OrangeBook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"orange_book"`
		Books struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"books"`
		Notebook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"notebook"`
		Ledger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ledger"`
		PageWithCurl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"page_with_curl"`
		Scroll struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scroll"`
		PageFacingUp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"page_facing_up"`
		Newspaper struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"newspaper"`
		RolledUpNewspaper struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rolled_up_newspaper"`
		BookmarkTabs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bookmark_tabs"`
		Bookmark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bookmark"`
		Label struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"label"`
		Moneybag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"moneybag"`
		Coin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coin"`
		Yen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yen"`
		Dollar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dollar"`
		Euro struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"euro"`
		Pound struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pound"`
		MoneyWithWings struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"money_with_wings"`
		CreditCard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"credit_card"`
		Receipt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"receipt"`
		Chart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chart"`
		Email struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"email"`
		EMail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"e-mail"`
		IncomingEnvelope struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"incoming_envelope"`
		EnvelopeWithArrow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"envelope_with_arrow"`
		OutboxTray struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"outbox_tray"`
		InboxTray struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"inbox_tray"`
		Package struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"package"`
		Mailbox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mailbox"`
		MailboxClosed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mailbox_closed"`
		MailboxWithMail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mailbox_with_mail"`
		MailboxWithNoMail struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mailbox_with_no_mail"`
		Postbox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"postbox"`
		BallotBoxWithBallot struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ballot_box_with_ballot"`
		Pencil2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pencil2"`
		BlackNib struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_nib"`
		LowerLeftFountainPen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lower_left_fountain_pen"`
		LowerLeftBallpointPen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lower_left_ballpoint_pen"`
		LowerLeftPaintbrush struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lower_left_paintbrush"`
		LowerLeftCrayon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lower_left_crayon"`
		Memo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"memo"`
		Briefcase struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"briefcase"`
		FileFolder struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"file_folder"`
		OpenFileFolder struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"open_file_folder"`
		CardIndexDividers struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"card_index_dividers"`
		Date struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"date"`
		Calendar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"calendar"`
		SpiralNotePad struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spiral_note_pad"`
		SpiralCalendarPad struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"spiral_calendar_pad"`
		CardIndex struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"card_index"`
		ChartWithUpwardsTrend struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chart_with_upwards_trend"`
		ChartWithDownwardsTrend struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chart_with_downwards_trend"`
		BarChart struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bar_chart"`
		Clipboard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"clipboard"`
		Pushpin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pushpin"`
		RoundPushpin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"round_pushpin"`
		Paperclip struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"paperclip"`
		LinkedPaperclips struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"linked_paperclips"`
		StraightRuler struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"straight_ruler"`
		TriangularRuler struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"triangular_ruler"`
		Scissors struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scissors"`
		CardFileBox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"card_file_box"`
		FileCabinet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"file_cabinet"`
		Wastebasket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wastebasket"`
		Lock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lock"`
		Unlock struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"unlock"`
		LockWithInkPen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lock_with_ink_pen"`
		ClosedLockWithKey struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"closed_lock_with_key"`
		Key struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"key"`
		OldKey struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"old_key"`
		Hammer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hammer"`
		Axe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"axe"`
		Pick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pick"`
		HammerAndPick struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hammer_and_pick"`
		HammerAndWrench struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hammer_and_wrench"`
		DaggerKnife struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dagger_knife"`
		CrossedSwords struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crossed_swords"`
		Gun struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gun"`
		Boomerang struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"boomerang"`
		BowAndArrow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bow_and_arrow"`
		Shield struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shield"`
		CarpentrySaw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"carpentry_saw"`
		Wrench struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wrench"`
		Screwdriver struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"screwdriver"`
		NutAndBolt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nut_and_bolt"`
		Gear struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gear"`
		Compression struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"compression"`
		Scales struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scales"`
		ProbingCane struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"probing_cane"`
		Link struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"link"`
		Chains struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chains"`
		Hook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hook"`
		Toolbox struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"toolbox"`
		Magnet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"magnet"`
		Ladder struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ladder"`
		Alembic struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"alembic"`
		TestTube struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"test_tube"`
		PetriDish struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"petri_dish"`
		Dna struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"dna"`
		Microscope struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"microscope"`
		Telescope struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"telescope"`
		SatelliteAntenna struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"satellite_antenna"`
		Syringe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"syringe"`
		DropOfBlood struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"drop_of_blood"`
		Pill struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pill"`
		AdhesiveBandage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"adhesive_bandage"`
		Crutch struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crutch"`
		Stethoscope struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"stethoscope"`
		XRay struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"x-ray"`
		Door struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"door"`
		Elevator struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"elevator"`
		Mirror struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mirror"`
		Window struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"window"`
		Bed struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bed"`
		CouchAndLamp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"couch_and_lamp"`
		Chair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"chair"`
		Toilet struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"toilet"`
		Plunger struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"plunger"`
		Shower struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shower"`
		Bathtub struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bathtub"`
		MouseTrap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mouse_trap"`
		Razor struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"razor"`
		LotionBottle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"lotion_bottle"`
		SafetyPin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"safety_pin"`
		Broom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"broom"`
		Basket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"basket"`
		RollOfPaper struct {
			Id       string        `json:"id"`
			Name     string        `json:"name"`
			Keywords []interface{} `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"roll_of_paper"`
		Bucket struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bucket"`
		Soap struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"soap"`
		Bubbles struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bubbles"`
		Toothbrush struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"toothbrush"`
		Sponge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sponge"`
		FireExtinguisher struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fire_extinguisher"`
		ShoppingTrolley struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"shopping_trolley"`
		Smoking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"smoking"`
		Coffin struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"coffin"`
		Headstone struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"headstone"`
		FuneralUrn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"funeral_urn"`
		Moyai struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"moyai"`
		Placard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"placard"`
		IdentificationCard struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"identification_card"`
		Atm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"atm"`
		PutLitterInItsPlace struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"put_litter_in_its_place"`
		PotableWater struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"potable_water"`
		Wheelchair struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wheelchair"`
		Mens struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mens"`
		Womens struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"womens"`
		Restroom struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"restroom"`
		BabySymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baby_symbol"`
		Wc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wc"`
		PassportControl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"passport_control"`
		Customs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"customs"`
		BaggageClaim struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"baggage_claim"`
		LeftLuggage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"left_luggage"`
		Warning struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"warning"`
		ChildrenCrossing struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"children_crossing"`
		NoEntry struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_entry"`
		NoEntrySign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_entry_sign"`
		NoBicycles struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_bicycles"`
		NoSmoking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_smoking"`
		DoNotLitter struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"do_not_litter"`
		NonPotableWater struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"non-potable_water"`
		NoPedestrians struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_pedestrians"`
		NoMobilePhones struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"no_mobile_phones"`
		Underage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"underage"`
		RadioactiveSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"radioactive_sign"`
		BiohazardSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"biohazard_sign"`
		ArrowUp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_up"`
		ArrowUpperRight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_upper_right"`
		ArrowRight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_right"`
		ArrowLowerRight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_lower_right"`
		ArrowDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_down"`
		ArrowLowerLeft struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_lower_left"`
		ArrowLeft struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_left"`
		ArrowUpperLeft struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_upper_left"`
		ArrowUpDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_up_down"`
		LeftRightArrow struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"left_right_arrow"`
		LeftwardsArrowWithHook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leftwards_arrow_with_hook"`
		ArrowRightHook struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_right_hook"`
		ArrowHeadingUp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_heading_up"`
		ArrowHeadingDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_heading_down"`
		ArrowsClockwise struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrows_clockwise"`
		ArrowsCounterclockwise struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrows_counterclockwise"`
		Back struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"back"`
		End struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"end"`
		On struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"on"`
		Soon struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"soon"`
		Top struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"top"`
		PlaceOfWorship struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"place_of_worship"`
		AtomSymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"atom_symbol"`
		OmSymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"om_symbol"`
		StarOfDavid struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"star_of_david"`
		WheelOfDharma struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wheel_of_dharma"`
		YinYang struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"yin_yang"`
		LatinCross struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"latin_cross"`
		OrthodoxCross struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"orthodox_cross"`
		StarAndCrescent struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"star_and_crescent"`
		PeaceSymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"peace_symbol"`
		MenorahWithNineBranches struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"menorah_with_nine_branches"`
		SixPointedStar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"six_pointed_star"`
		Aries struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"aries"`
		Taurus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"taurus"`
		Gemini struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gemini"`
		Cancer struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cancer"`
		Leo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"leo"`
		Virgo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"virgo"`
		Libra struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"libra"`
		Scorpius struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"scorpius"`
		Sagittarius struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sagittarius"`
		Capricorn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"capricorn"`
		Aquarius struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"aquarius"`
		Pisces struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pisces"`
		Ophiuchus struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ophiuchus"`
		TwistedRightwardsArrows struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"twisted_rightwards_arrows"`
		Repeat struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"repeat"`
		RepeatOne struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"repeat_one"`
		ArrowForward struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_forward"`
		FastForward struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fast_forward"`
		BlackRightPointingDoubleTriangleWithVerticalBar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_right_pointing_double_triangle_with_vertical_bar"`
		BlackRightPointingTriangleWithDoubleVerticalBar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_right_pointing_triangle_with_double_vertical_bar"`
		ArrowBackward struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_backward"`
		Rewind struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rewind"`
		BlackLeftPointingDoubleTriangleWithVerticalBar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_left_pointing_double_triangle_with_vertical_bar"`
		ArrowUpSmall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_up_small"`
		ArrowDoubleUp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_double_up"`
		ArrowDownSmall struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_down_small"`
		ArrowDoubleDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"arrow_double_down"`
		DoubleVerticalBar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"double_vertical_bar"`
		BlackSquareForStop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_square_for_stop"`
		BlackCircleForRecord struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_circle_for_record"`
		Eject struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eject"`
		Cinema struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cinema"`
		LowBrightness struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"low_brightness"`
		HighBrightness struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"high_brightness"`
		SignalStrength struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"signal_strength"`
		VibrationMode struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"vibration_mode"`
		MobilePhoneOff struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"mobile_phone_off"`
		FemaleSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"female_sign"`
		MaleSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"male_sign"`
		TransgenderSymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"transgender_symbol"`
		HeavyMultiplicationX struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_multiplication_x"`
		HeavyPlusSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_plus_sign"`
		HeavyMinusSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_minus_sign"`
		HeavyDivisionSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_division_sign"`
		HeavyEqualsSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_equals_sign"`
		Infinity struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"infinity"`
		Bangbang struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"bangbang"`
		Interrobang struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"interrobang"`
		Question struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"question"`
		GreyQuestion struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grey_question"`
		GreyExclamation struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"grey_exclamation"`
		Exclamation struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"exclamation"`
		WavyDash struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"wavy_dash"`
		CurrencyExchange struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"currency_exchange"`
		HeavyDollarSign struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_dollar_sign"`
		MedicalSymbol struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"medical_symbol"`
		Recycle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"recycle"`
		FleurDeLis struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fleur_de_lis"`
		Trident struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"trident"`
		NameBadge struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"name_badge"`
		Beginner struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"beginner"`
		O struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"o"`
		WhiteCheckMark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_check_mark"`
		BallotBoxWithCheck struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ballot_box_with_check"`
		HeavyCheckMark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"heavy_check_mark"`
		X struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"x"`
		NegativeSquaredCrossMark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"negative_squared_cross_mark"`
		CurlyLoop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"curly_loop"`
		Loop struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"loop"`
		PartAlternationMark struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"part_alternation_mark"`
		EightSpokedAsterisk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eight_spoked_asterisk"`
		EightPointedBlackStar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eight_pointed_black_star"`
		Sparkle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sparkle"`
		Copyright struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"copyright"`
		Registered struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"registered"`
		Tm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"tm"`
		Hash struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"hash"`
		KeycapStar struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"keycap_star"`
		Zero struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"zero"`
		One struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"one"`
		Two struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"two"`
		Three struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"three"`
		Four struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"four"`
		Five struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"five"`
		Six struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"six"`
		Seven struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"seven"`
		Eight struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"eight"`
		Nine struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"nine"`
		KeycapTen struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"keycap_ten"`
		CapitalAbcd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"capital_abcd"`
		Abcd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"abcd"`
		Symbols struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"symbols"`
		Abc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"abc"`
		A struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"a"`
		Ab struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ab"`
		B struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"b"`
		Cl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cl"`
		Cool struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cool"`
		Free struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"free"`
		InformationSource struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"information_source"`
		Id struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"id"`
		M struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"m"`
		New struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"new"`
		Ng struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ng"`
		O2 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"o2"`
		Ok struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ok"`
		Parking struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"parking"`
		Sos struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sos"`
		Up struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"up"`
		Vs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"vs"`
		Koko struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"koko"`
		Sa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"sa"`
		U6708 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u6708"`
		U6709 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u6709"`
		U6307 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u6307"`
		IdeographAdvantage struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ideograph_advantage"`
		U5272 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u5272"`
		U7121 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u7121"`
		U7981 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u7981"`
		Accept struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"accept"`
		U7533 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u7533"`
		U5408 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u5408"`
		U7A7A struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u7a7a"`
		Congratulations struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"congratulations"`
		Secret struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"secret"`
		U55B6 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u55b6"`
		U6E80 struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"u6e80"`
		RedCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"red_circle"`
		LargeOrangeCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_orange_circle"`
		LargeYellowCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_yellow_circle"`
		LargeGreenCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_green_circle"`
		LargeBlueCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_blue_circle"`
		LargePurpleCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_purple_circle"`
		LargeBrownCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_brown_circle"`
		BlackCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_circle"`
		WhiteCircle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_circle"`
		LargeRedSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_red_square"`
		LargeOrangeSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_orange_square"`
		LargeYellowSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_yellow_square"`
		LargeGreenSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_green_square"`
		LargeBlueSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_blue_square"`
		LargePurpleSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_purple_square"`
		LargeBrownSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_brown_square"`
		BlackLargeSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_large_square"`
		WhiteLargeSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_large_square"`
		BlackMediumSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_medium_square"`
		WhiteMediumSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_medium_square"`
		BlackMediumSmallSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_medium_small_square"`
		WhiteMediumSmallSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_medium_small_square"`
		BlackSmallSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_small_square"`
		WhiteSmallSquare struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_small_square"`
		LargeOrangeDiamond struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_orange_diamond"`
		LargeBlueDiamond struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"large_blue_diamond"`
		SmallOrangeDiamond struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"small_orange_diamond"`
		SmallBlueDiamond struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"small_blue_diamond"`
		SmallRedTriangle struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"small_red_triangle"`
		SmallRedTriangleDown struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"small_red_triangle_down"`
		DiamondShapeWithADotInside struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"diamond_shape_with_a_dot_inside"`
		RadioButton struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"radio_button"`
		WhiteSquareButton struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"white_square_button"`
		BlackSquareButton struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"black_square_button"`
		CheckeredFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"checkered_flag"`
		TriangularFlagOnPost struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"triangular_flag_on_post"`
		CrossedFlags struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"crossed_flags"`
		WavingBlackFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waving_black_flag"`
		WavingWhiteFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"waving_white_flag"`
		RainbowFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"rainbow-flag"`
		TransgenderFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"transgender_flag"`
		PirateFlag struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"pirate_flag"`
		FlagAc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ac"`
		FlagAd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ad"`
		FlagAe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ae"`
		FlagAf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-af"`
		FlagAg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ag"`
		FlagAi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ai"`
		FlagAl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-al"`
		FlagAm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-am"`
		FlagAo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ao"`
		FlagAq struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-aq"`
		FlagAr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ar"`
		FlagAs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-as"`
		FlagAt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-at"`
		FlagAu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-au"`
		FlagAw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-aw"`
		FlagAx struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ax"`
		FlagAz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-az"`
		FlagBa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ba"`
		FlagBb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bb"`
		FlagBd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bd"`
		FlagBe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-be"`
		FlagBf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bf"`
		FlagBg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bg"`
		FlagBh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bh"`
		FlagBi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bi"`
		FlagBj struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bj"`
		FlagBl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bl"`
		FlagBm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bm"`
		FlagBn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bn"`
		FlagBo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bo"`
		FlagBq struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bq"`
		FlagBr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-br"`
		FlagBs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bs"`
		FlagBt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bt"`
		FlagBv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bv"`
		FlagBw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bw"`
		FlagBy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-by"`
		FlagBz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-bz"`
		FlagCa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ca"`
		FlagCc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cc"`
		FlagCd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cd"`
		FlagCf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cf"`
		FlagCg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cg"`
		FlagCh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ch"`
		FlagCi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ci"`
		FlagCk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ck"`
		FlagCl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cl"`
		FlagCm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cm"`
		Cn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"cn"`
		FlagCo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-co"`
		FlagCp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cp"`
		FlagCr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cr"`
		FlagCu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cu"`
		FlagCv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cv"`
		FlagCw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cw"`
		FlagCx struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cx"`
		FlagCy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cy"`
		FlagCz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-cz"`
		De struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"de"`
		FlagDg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-dg"`
		FlagDj struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-dj"`
		FlagDk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-dk"`
		FlagDm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-dm"`
		FlagDo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-do"`
		FlagDz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-dz"`
		FlagEa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ea"`
		FlagEc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ec"`
		FlagEe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ee"`
		FlagEg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-eg"`
		FlagEh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-eh"`
		FlagEr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-er"`
		Es struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"es"`
		FlagEt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-et"`
		FlagEu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-eu"`
		FlagFi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-fi"`
		FlagFj struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-fj"`
		FlagFk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-fk"`
		FlagFm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-fm"`
		FlagFo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-fo"`
		Fr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"fr"`
		FlagGa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ga"`
		Gb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"gb"`
		FlagGd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gd"`
		FlagGe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ge"`
		FlagGf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gf"`
		FlagGg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gg"`
		FlagGh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gh"`
		FlagGi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gi"`
		FlagGl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gl"`
		FlagGm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gm"`
		FlagGn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gn"`
		FlagGp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gp"`
		FlagGq struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gq"`
		FlagGr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gr"`
		FlagGs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gs"`
		FlagGt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gt"`
		FlagGu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gu"`
		FlagGw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gw"`
		FlagGy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-gy"`
		FlagHk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-hk"`
		FlagHm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-hm"`
		FlagHn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-hn"`
		FlagHr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-hr"`
		FlagHt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ht"`
		FlagHu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-hu"`
		FlagIc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ic"`
		FlagId struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-id"`
		FlagIe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ie"`
		FlagIl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-il"`
		FlagIm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-im"`
		FlagIn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-in"`
		FlagIo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-io"`
		FlagIq struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-iq"`
		FlagIr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ir"`
		FlagIs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-is"`
		It struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"it"`
		FlagJe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-je"`
		FlagJm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-jm"`
		FlagJo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-jo"`
		Jp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"jp"`
		FlagKe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ke"`
		FlagKg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kg"`
		FlagKh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kh"`
		FlagKi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ki"`
		FlagKm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-km"`
		FlagKn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kn"`
		FlagKp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kp"`
		Kr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"kr"`
		FlagKw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kw"`
		FlagKy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ky"`
		FlagKz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-kz"`
		FlagLa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-la"`
		FlagLb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lb"`
		FlagLc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lc"`
		FlagLi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-li"`
		FlagLk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lk"`
		FlagLr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lr"`
		FlagLs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ls"`
		FlagLt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lt"`
		FlagLu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lu"`
		FlagLv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-lv"`
		FlagLy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ly"`
		FlagMa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ma"`
		FlagMc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mc"`
		FlagMd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-md"`
		FlagMe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-me"`
		FlagMf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mf"`
		FlagMg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mg"`
		FlagMh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mh"`
		FlagMk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mk"`
		FlagMl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ml"`
		FlagMm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mm"`
		FlagMn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mn"`
		FlagMo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mo"`
		FlagMp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mp"`
		FlagMq struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mq"`
		FlagMr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mr"`
		FlagMs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ms"`
		FlagMt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mt"`
		FlagMu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mu"`
		FlagMv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mv"`
		FlagMw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mw"`
		FlagMx struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mx"`
		FlagMy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-my"`
		FlagMz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-mz"`
		FlagNa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-na"`
		FlagNc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nc"`
		FlagNe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ne"`
		FlagNf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nf"`
		FlagNg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ng"`
		FlagNi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ni"`
		FlagNl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nl"`
		FlagNo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-no"`
		FlagNp struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-np"`
		FlagNr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nr"`
		FlagNu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nu"`
		FlagNz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-nz"`
		FlagOm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-om"`
		FlagPa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pa"`
		FlagPe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pe"`
		FlagPf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pf"`
		FlagPg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pg"`
		FlagPh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ph"`
		FlagPk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pk"`
		FlagPl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pl"`
		FlagPm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pm"`
		FlagPn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pn"`
		FlagPr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pr"`
		FlagPs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ps"`
		FlagPt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pt"`
		FlagPw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-pw"`
		FlagPy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-py"`
		FlagQa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-qa"`
		FlagRe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-re"`
		FlagRo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ro"`
		FlagRs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-rs"`
		Ru struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"ru"`
		FlagRw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-rw"`
		FlagSa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sa"`
		FlagSb struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sb"`
		FlagSc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sc"`
		FlagSd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sd"`
		FlagSe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-se"`
		FlagSg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sg"`
		FlagSh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sh"`
		FlagSi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-si"`
		FlagSj struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sj"`
		FlagSk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sk"`
		FlagSl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sl"`
		FlagSm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sm"`
		FlagSn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sn"`
		FlagSo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-so"`
		FlagSr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sr"`
		FlagSs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ss"`
		FlagSt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-st"`
		FlagSv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sv"`
		FlagSx struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sx"`
		FlagSy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sy"`
		FlagSz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-sz"`
		FlagTa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ta"`
		FlagTc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tc"`
		FlagTd struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-td"`
		FlagTf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tf"`
		FlagTg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tg"`
		FlagTh struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-th"`
		FlagTj struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tj"`
		FlagTk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tk"`
		FlagTl struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tl"`
		FlagTm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tm"`
		FlagTn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tn"`
		FlagTo struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-to"`
		FlagTr struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tr"`
		FlagTt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tt"`
		FlagTv struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tv"`
		FlagTw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tw"`
		FlagTz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-tz"`
		FlagUa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ua"`
		FlagUg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ug"`
		FlagUm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-um"`
		FlagUn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-un"`
		Us struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"us"`
		FlagUy struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-uy"`
		FlagUz struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-uz"`
		FlagVa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-va"`
		FlagVc struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-vc"`
		FlagVe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ve"`
		FlagVg struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-vg"`
		FlagVi struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-vi"`
		FlagVn struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-vn"`
		FlagVu struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-vu"`
		FlagWf struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-wf"`
		FlagWs struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ws"`
		FlagXk struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-xk"`
		FlagYe struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-ye"`
		FlagYt struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-yt"`
		FlagZa struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-za"`
		FlagZm struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-zm"`
		FlagZw struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-zw"`
		FlagEngland struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-england"`
		FlagScotland struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-scotland"`
		FlagWales struct {
			Id       string   `json:"id"`
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
			Skins    []struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			} `json:"skins"`
			Version int `json:"version"`
		} `json:"flag-wales"`
	} `json:"emojis"`
	Aliases struct {
		Satisfied                                      string `json:"satisfied"`
		GrinningFaceWithStarEyes                       string `json:"grinning_face_with_star_eyes"`
		GrinningFaceWithOneLargeAndOneSmallEye         string `json:"grinning_face_with_one_large_and_one_small_eye"`
		SmilingFaceWithSmilingEyesAndHandCoveringMouth string `json:"smiling_face_with_smiling_eyes_and_hand_covering_mouth"`
		FaceWithFingerCoveringClosedLips               string `json:"face_with_finger_covering_closed_lips"`
		FaceWithOneEyebrowRaised                       string `json:"face_with_one_eyebrow_raised"`
		FaceWithOpenMouthVomiting                      string `json:"face_with_open_mouth_vomiting"`
		ShockedFaceWithExplodingHead                   string `json:"shocked_face_with_exploding_head"`
		SeriousFaceWithSymbolsCoveringMouth            string `json:"serious_face_with_symbols_covering_mouth"`
		Poop                                           string `json:"poop"`
		Shit                                           string `json:"shit"`
		Collision                                      string `json:"collision"`
		RaisedHand                                     string `json:"raised_hand"`
		HandWithIndexAndMiddleFingersCrossed           string `json:"hand_with_index_and_middle_fingers_crossed"`
		SignOfTheHorns                                 string `json:"sign_of_the_horns"`
		ReversedHandWithMiddleFingerExtended           string `json:"reversed_hand_with_middle_finger_extended"`
		Thumbsup                                       string `json:"thumbsup"`
		Thumbsdown                                     string `json:"thumbsdown"`
		Punch                                          string `json:"punch"`
		MotherChristmas                                string `json:"mother_christmas"`
		Running                                        string `json:"running"`
		ManWithBunnyEarsPartying                       string `json:"man-with-bunny-ears-partying"`
		WomanWithBunnyEarsPartying                     string `json:"woman-with-bunny-ears-partying"`
		WomenHoldingHands                              string `json:"women_holding_hands"`
		WomanAndManHoldingHands                        string `json:"woman_and_man_holding_hands"`
		Couple                                         string `json:"couple"`
		MenHoldingHands                                string `json:"men_holding_hands"`
		PawPrints                                      string `json:"paw_prints"`
		Flipper                                        string `json:"flipper"`
		Honeybee                                       string `json:"honeybee"`
		LadyBeetle                                     string `json:"lady_beetle"`
		Cooking                                        string `json:"cooking"`
		Knife                                          string `json:"knife"`
		RedCar                                         string `json:"red_car"`
		Sailboat                                       string `json:"sailboat"`
		WaxingGibbousMoon                              string `json:"waxing_gibbous_moon"`
		SunSmallCloud                                  string `json:"sun_small_cloud"`
		SunBehindCloud                                 string `json:"sun_behind_cloud"`
		SunBehindRainCloud                             string `json:"sun_behind_rain_cloud"`
		LightningCloud                                 string `json:"lightning_cloud"`
		TornadoCloud                                   string `json:"tornado_cloud"`
		Tshirt                                         string `json:"tshirt"`
		Shoe                                           string `json:"shoe"`
		Telephone                                      string `json:"telephone"`
		Lantern                                        string `json:"lantern"`
		OpenBook                                       string `json:"open_book"`
		Envelope                                       string `json:"envelope"`
		Pencil                                         string `json:"pencil"`
		HeavyExclamationMark                           string `json:"heavy_exclamation_mark"`
		StaffOfAesculapius                             string `json:"staff_of_aesculapius"`
		FlagCn                                         string `json:"flag-cn"`
		FlagDe                                         string `json:"flag-de"`
		FlagEs                                         string `json:"flag-es"`
		FlagFr                                         string `json:"flag-fr"`
		Uk                                             string `json:"uk"`
		FlagGb                                         string `json:"flag-gb"`
		FlagIt                                         string `json:"flag-it"`
		FlagJp                                         string `json:"flag-jp"`
		FlagKr                                         string `json:"flag-kr"`
		FlagRu                                         string `json:"flag-ru"`
		FlagUs                                         string `json:"flag-us"`
	} `json:"aliases"`
	Sheet struct {
		Cols int `json:"cols"`
		Rows int `json:"rows"`
	} `json:"sheet"`
}
