////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"bytes"
	"math/rand"
	"testing"
	"gitlab.com/elixxir/primitives/format"
)

func randomString(seed int64, length int) []byte {
	buffer := make([]byte, length)
	rand.Seed(seed)
	rand.Read(buffer)
	return buffer
}

// Partitioning an empty message should result in one byte slice with just
// the front matter.
func TestPartitionEmptyMessage(t *testing.T) {
	id := []byte{0x05}
	actual, err := Partition(randomString(0, 0), id)
	if err != nil {
		t.Error(err.Error())
	}
	expected := [][]byte{{0x05, 0x0, 0x0}}
	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition empty message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

// Partitioning a short message should result in one partition that includes
// the short message.
func TestPartitionShort(t *testing.T) {
	id := []byte{0x03}
	randomBytes := randomString(0, 50)
	actual, err := Partition(randomBytes, id)
	if err != nil {
		t.Error(err.Error())
	}
	expected := [][]byte{{0x03, 0x0, 0x0}}
	expected[0] = append(expected[0], randomBytes...)
	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition short message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

// Partitioning a longer message should result in more than one partition that,
// in sum, contains the whole message.
func TestPartitionLong(t *testing.T) {
	id := []byte{0xa2, 0x54}
	randomBytes := randomString(0, 300)
	actual, err := Partition(randomBytes, id)

	if err != nil {
		t.Error(err.Error())
	}

	expected := make([][]byte, 2)
	// id
	expected[0] = append(expected[0], id...)
	// index
	expected[0] = append(expected[0], 0, 1)
	// part of random string
	expected[0] = append(expected[0], randomBytes[:format.DATA_LEN-4]...)

	// id
	expected[1] = append(expected[1], id...)
	// index
	expected[1] = append(expected[1], 1, 1)
	// other part of random string
	expected[1] = append(expected[1], randomBytes[format.DATA_LEN-4:]...)

	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition long message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

// Due to the data types I used to fill out the front matter, there's a limit to
// how many parts a message can be for one multi-part message. This test makes
// sure that the indexes grow as expected to fill the whole space.
func TestPartitionLongest(t *testing.T) {
	// I'm assuming that 5 bytes will be the longest possible ID because that
	// is the max length of a uvarint with 32 bits
	id := []byte{0x1f, 0x2f, 0x3f, 0x4f, 0x5f}
	actual, err := Partition(randomString(0, 51199), id)

	if err != nil {
		t.Error(err.Error())
	}

	expectedNumberOfPartitions := 256

	if len(actual) != expectedNumberOfPartitions {
		t.Errorf("Expected a 51199-byte message to split into %v partitions",
			expectedNumberOfPartitions)
	}

	// check the index and max index of the last partition
	expectedIdx := byte(255)
	idxLocation := len(id)
	maxIdxLocation := len(id) + 1
	actualIdx := actual[len(actual)-1][idxLocation]
	actualMaxIdx := actual[len(actual)-1][maxIdxLocation]
	if actualIdx != expectedIdx {
		t.Errorf("Expected index of %v on the last partition, got %v",
			expectedIdx, actualIdx)
	}
	if actualMaxIdx != expectedIdx {
		t.Errorf("Expected max index of %v on the last partition, got %v",
			expectedIdx, actualMaxIdx)
	}
}

// Tests production of the error that occurs when you ask to partition a
// message that's too long to partition
func TestPartitionTooLong(t *testing.T) {
	id := []byte{0x1f, 0x2f, 0x3f, 0x4f, 0x5f}
	_, err := Partition(randomString(0, 57856), id)

	if err == nil {
		t.Error("Partition() processed a message that was too long to be" +
			" partitioned")
	}
}

// Tests Assemble with a synthetic test case, without invoking Partition.
func TestOnlyAssemble(t *testing.T) {
	messageChunks := []string{"Han Singular, ", "my child, ",
		"awaken and embrace ", "the glory that is", " your birthright."}

	completeMessage := ""
	for i := range messageChunks {
		completeMessage += messageChunks[i]
	}

	partitions := make([][]byte, len(messageChunks))
	for i := range partitions {
		partitions[i] = append(partitions[i], messageChunks[i]...)
	}

	if completeMessage != string(Assemble(partitions)) {
		t.Errorf("TestOnlyAssemble: got \"%v\"; expected \"%v\".",
			string(Assemble(partitions)), completeMessage)
	}
}

// This tests the pipeline end-to-end, making sure that the same text that goes
// into partitioning can come out of it intact.
func TestAssembleAndPartition(t *testing.T) {
	expected := []string{
		"short message",
		// 5008 bytes
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Fusce tristique neque sed diam efficitur pulvinar. Proin posuere tortor id sodales elementum. Ut nec viverra libero. Proin et dui consequat nulla rhoncus facilisis. Phasellus semper at tortor ut ullamcorper. Aliquam accumsan auctor elit, vel tincidunt nulla bibendum et. Integer dictum ligula mauris, sit amet dignissim quam ornare sed. Mauris diam orci, ultrices vitae tellus non, faucibus scelerisque ante. Morbi fringilla massa purus, eu fringilla eros ultricies vel. Suspendisse nisi nisl, interdum quis porttitor quis, facilisis ac mauris. Integer in pretium erat, sed egestas quam. Donec eleifend felis dapibus mauris ullamcorper feugiat. Nulla at pharetra lectus. Pellentesque libero metus, efficitur at venenatis non, pharetra eu nisl. Donec id lorem dignissim, euismod elit vel, efficitur lacus. In finibus, orci ut rhoncus mollis, sem ex aliquet nunc, sed pretium eros justo ac tortor. Etiam vehicula dapibus lectus sed condimentum. Cras porta nulla sit amet pretium suscipit. Vivamus vestibulum sed nibh non vestibulum. Suspendisse sit amet purus at sapien mollis sollicitudin eu id turpis. Nulla dapibus in urna sit amet luctus. Proin faucibus quis dui porta volutpat. Duis sed ultrices lacus. Integer interdum finibus sem, in finibus urna eleifend at. Curabitur urna mi, auctor et ligula a, tristique pretium ex. Vivamus vitae felis non nunc rhoncus mattis. Integer fringilla volutpat lorem ac dictum. Praesent sed nibh et purus sollicitudin iaculis at eu metus. Nunc lobortis fermentum magna, quis varius velit blandit vel. Quisque fringilla lacinia magna ac euismod. Vestibulum velit ipsum, bibendum sagittis leo sed, pretium porta magna. Nulla facilisi. Aenean elementum posuere consequat. Cras placerat vulputate magna, at condimentum nibh sagittis quis. Pellentesque auctor tortor vehicula ante tristique, in auctor purus efficitur. Vivamus sapien lorem, viverra ut lacinia at, laoreet nec diam. Proin finibus, elit ac ultricies fermentum, eros erat imperdiet lacus, sed laoreet dui elit sed odio. Etiam id hendrerit quam, quis rhoncus mauris. Proin ac ante bibendum, malesuada mauris vitae, tempor quam. Nulla vitae pulvinar nunc. Vestibulum quis vulputate risus, et gravida enim. Sed tellus lacus, sagittis sit amet sodales non, varius ultrices massa. Aliquam nec volutpat sem. Nam porttitor, nibh vitae iaculis posuere, magna ante placerat elit, ut suscipit odio dui vitae libero. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam tincidunt tellus risus, eget laoreet odio suscipit et. Nulla scelerisque interdum sapien, in tempus mi malesuada vel. Donec et urna sit amet purus pulvinar tincidunt. Mauris fermentum quis lacus at scelerisque. Interdum et malesuada fames ac ante ipsum primis in faucibus. Aenean viverra erat vel sagittis blandit. Praesent in purus sed tortor consequat vehicula. Mauris non iaculis diam, nec vestibulum nisl. Phasellus arcu mi, luctus id felis sit amet, feugiat pellentesque tortor. Curabitur at dui dolor. Nunc semper quam pharetra, suscipit quam at, fringilla justo. In feugiat ipsum eu lectus aliquet ultrices. Curabitur fringilla tincidunt vehicula. Donec laoreet facilisis ante ac maximus. Aliquam lectus diam, pulvinar quis arcu in, molestie tincidunt quam. Sed aliquet orci id arcu finibus congue. Ut nulla lacus, dictum eget sem in, condimentum mattis massa. Donec suscipit, sapien nec euismod tincidunt, velit lectus iaculis ligula, sed sagittis tellus odio at nisl. Aenean mattis tellus in convallis aliquet. Duis posuere, augue id pellentesque accumsan, enim orci congue diam, a venenatis metus tellus id nibh. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Sed ex massa, lobortis sed risus in, blandit tincidunt enim. Suspendisse fringilla lacinia velit sit amet varius. Donec ac malesuada nisl, vel sagittis mauris. Sed eu blandit orci. Ut porta orci sed dui blandit tristique. Donec ac tellus et nisl fermentum volutpat. Nullam ipsum mi, aliquet ut mattis non, imperdiet non massa. Phasellus tincidunt mauris ac convallis convallis. Nunc blandit velit vel fermentum rhoncus. Nam dictum mi in fringilla semper. Nunc tristique congue velit et cursus. Vivamus rhoncus porta lacus posuere sodales. Quisque in interdum lectus, in imperdiet lacus. Proin vel arcu non arcu commodo rhoncus ac rhoncus velit. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Nulla eget massa et nibh suscipit egestas nec a justo. Morbi a semper diam, vitae rutrum odio. Nunc a nisl quam. Proin ornare luctus sem, et rhoncus est mattis in. Donec hendrerit, augue id sodales maximus, justo magna faucibus libero, eu hendrerit diam elit vel massa. Nulla dictum purus nisi, eget varius dui lacinia non. Fusce ut mauris ut massa imperdiet consequat. Proin id eros vitae odio gravida convallis. Donec faucibus, massa quis volutpat. `,
		// Near max length of multi-part messages
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc ipsum odio, suscipit nec tempor vitae, pretium convallis felis. Integer rutrum dolor sit amet tellus semper volutpat. Mauris eleifend massa ac iaculis aliquam. Ut ac urna faucibus, commodo risus vitae, eleifend urna. Duis nec diam quam. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Vestibulum aliquam at elit ut eleifend. Phasellus id massa malesuada, elementum ante sed, porta velit. In finibus a nulla at maximus. Suspendisse condimentum consequat volutpat. Cras varius volutpat dapibus. Curabitur venenatis semper varius. Etiam condimentum ligula diam, eu commodo purus placerat id. Morbi ut ipsum ac elit egestas tempor.

Ut consectetur elementum orci, nec feugiat sapien tincidunt eget. Vivamus tincidunt ut massa nec imperdiet. Suspendisse potenti. Donec venenatis iaculis libero, vel facilisis diam dictum vel. Nam id orci turpis. Integer nec turpis id nunc consequat ornare. Aenean rhoncus interdum tortor, ut suscipit leo faucibus ornare. Etiam fringilla neque sit amet dolor ullamcorper rutrum. Quisque pharetra maximus nibh quis tincidunt. Phasellus eu risus nisi. Nunc vitae viverra tellus, id porta nisi. Proin condimentum fringilla risus, ut placerat arcu imperdiet eget. Nunc nec interdum est. Sed semper, purus non cursus interdum, turpis tellus ultricies felis, nec venenatis velit nunc at lacus.

Phasellus et eleifend quam, sit amet sagittis sem. Mauris in sagittis quam, non viverra metus. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Sed vitae bibendum quam, id convallis nisi. Sed ultricies malesuada enim, non posuere mi tempor at. Nam quis ante quis leo rhoncus semper vel at metus. Vestibulum vitae viverra leo, nec luctus ante. Maecenas sed elementum purus. Interdum et malesuada fames ac ante ipsum primis in faucibus. Sed ornare odio ut mi tempor mattis.

Mauris a ex sem. Praesent efficitur lacinia ligula et rhoncus. Quisque id nunc urna. Nulla laoreet urna suscipit, varius quam eget, consectetur est. Donec suscipit, quam vestibulum vehicula viverra, ipsum purus aliquam magna, sed elementum leo nulla sed augue. Curabitur semper non ligula sed varius. Vivamus nec nulla dolor. Sed ornare commodo mollis. Pellentesque augue ipsum, imperdiet id laoreet non, aliquam id ex. Donec vel enim nibh.

Vivamus accumsan, nunc sit amet lacinia condimentum, metus libero mollis velit, sit amet gravida nulla ex vel orci. Quisque gravida, nisi vel molestie ultrices, mauris dolor viverra lacus, at egestas turpis nibh tincidunt elit. Praesent luctus nunc ut mollis malesuada. Integer mi risus, dapibus tristique justo ac, malesuada commodo felis. Donec mollis tincidunt ex sed bibendum. Curabitur vitae blandit lorem, nec finibus turpis. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin mattis, metus in porttitor tincidunt, nibh ligula lacinia tellus, vel finibus mauris sapien eget odio. Interdum et malesuada fames ac ante ipsum primis in faucibus. Quisque mattis velit sed nunc tincidunt iaculis. Nam commodo nulla ac diam semper tempus. Integer risus libero, aliquam vitae lectus vitae, fermentum efficitur neque. Praesent tempor nisi nisl, quis rutrum mi sodales a. Phasellus quam est, sagittis eu bibendum sed, eleifend non ipsum. Sed magna tortor, sollicitudin a sem ac, bibendum cursus eros. Aliquam dictum risus quis felis consectetur tempor.

Donec elementum, velit a consectetur euismod, mi enim lobortis odio, id sodales nulla odio efficitur justo. Aliquam erat volutpat. Integer nisl ex, tincidunt fermentum vehicula sed, malesuada non dui. Proin a lacinia diam. Quisque et eros gravida, ultricies dui vel, cursus tellus. Pellentesque eget dictum orci. Fusce metus lectus, eleifend eget tempus non, laoreet quis dolor. Etiam dui odio, blandit vel erat nec, ultricies molestie nulla. Etiam urna enim, imperdiet vestibulum dolor in, lobortis aliquam mi. Maecenas tincidunt sed lorem sed malesuada. Suspendisse elementum rutrum massa, sit amet bibendum metus tincidunt sed. Quisque et euismod metus.

Pellentesque auctor, risus sit amet pellentesque tempor, odio metus sodales tortor, commodo blandit nibh nisi eget sem. Nunc non mattis erat, sit amet imperdiet justo. Vestibulum commodo nisl id facilisis eleifend. Duis a nulla sapien. Donec vulputate lobortis odio. Aliquam in enim porttitor, dictum ante id, condimentum orci. In hac habitasse platea dictumst.

Vestibulum sollicitudin sollicitudin urna, non suscipit dolor tristique ac. Nulla congue non odio non elementum. Integer id leo diam. Curabitur sed massa mi. Curabitur maximus elit mi, nec placerat felis scelerisque eu. Maecenas vitae ante ex. Sed interdum suscipit tortor. Integer semper dolor nisl, sed tempus dui pulvinar ornare. Curabitur lobortis orci nibh, vitae pharetra est imperdiet non. Mauris sodales vehicula neque id hendrerit. Etiam tincidunt elit metus, vitae sollicitudin mi aliquet non. Ut faucibus eleifend vulputate. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Integer ut tincidunt turpis, eu dictum elit. Etiam bibendum fringilla est. Nullam sed congue quam.

Phasellus dapibus leo est, id elementum dolor tristique in. Duis vitae porttitor lectus. Vivamus convallis tortor ex, sed ullamcorper est interdum sed. Quisque ante quam, scelerisque at efficitur eget, facilisis ut mi. Vestibulum sit amet facilisis turpis. Fusce interdum gravida efficitur. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas tincidunt sagittis pulvinar. Suspendisse imperdiet, justo ut aliquet posuere, mauris odio tincidunt purus, id finibus erat lacus in nisi. Nam vitae consectetur risus. Etiam diam lorem, maximus vel vestibulum quis, lobortis in mi. Nulla eu sagittis lectus.

Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Aliquam erat volutpat. Morbi sollicitudin aliquam tellus, sed porttitor lorem imperdiet quis. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Proin nulla ipsum, sagittis et egestas in, egestas eget mauris. Etiam maximus eros sit amet luctus elementum. Praesent nibh lacus, ultrices vel sodales vitae, rhoncus vestibulum nulla. Duis accumsan lacinia dui sit amet convallis. Nullam massa neque, varius eu orci a, pretium porttitor risus. Fusce mollis ante sit amet velit sollicitudin hendrerit. Morbi eu iaculis ante. Aliquam laoreet nisi a ex dignissim lobortis in vel felis. Maecenas sagittis nibh ut purus tempor, at elementum velit commodo. Fusce ac neque pharetra dolor varius facilisis.

Nam ex purus, luctus sit amet lorem vitae, volutpat auctor tellus. Fusce euismod nibh non ex vestibulum, in iaculis justo placerat. Vivamus vulputate interdum molestie. Nulla pretium tempus nulla nec fermentum. Morbi blandit, metus non suscipit maximus, massa nisi sodales leo, eleifend ornare ex dolor ut justo. Sed pellentesque nec ex vitae placerat. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Quisque vel magna non tortor vulputate tristique.

Sed facilisis libero nec arcu cursus pulvinar. Quisque ipsum elit, porttitor non risus porta, tincidunt mattis ex. Fusce convallis ligula eget consectetur fringilla. Quisque fermentum, leo at porttitor semper, est elit fermentum mi, vitae efficitur felis magna ac mauris. Quisque ut efficitur lectus. Proin consequat imperdiet odio. Nulla dui lorem, viverra a dapibus laoreet, lacinia in libero. Suspendisse congue, diam ac facilisis hendrerit, tortor lorem sollicitudin diam, in dapibus lacus erat vel arcu. Praesent dolor nisl, commodo ut condimentum quis, mattis sed risus. Vivamus a ante et neque efficitur aliquet.

Nunc eget blandit nisi. Nullam pretium felis ac neque fermentum fermentum. Proin varius nulla ex, ut vulputate neque consequat quis. Etiam ac enim egestas felis luctus dapibus. Nullam vel eros eget dui euismod vestibulum. Vestibulum interdum quam libero, id blandit quam efficitur vitae. Integer viverra iaculis nibh consectetur euismod. Integer laoreet eu risus sed venenatis. Ut quis neque tortor. Maecenas vulputate magna eu mi tincidunt cursus eget vitae elit. Nam a mi hendrerit, porta orci in, dignissim est.

Phasellus sit amet ligula nec lorem commodo dignissim. Sed gravida, augue eu pharetra bibendum, augue justo ultrices sapien, molestie tempor quam mauris eget ipsum. Ut tristique tristique arcu, elementum pulvinar sapien aliquam a. Aliquam nec eros faucibus, laoreet massa eu, imperdiet velit. Suspendisse finibus nisl ac tellus fringilla, eget aliquet sem dapibus. Ut quam turpis, aliquam ac fringilla non, imperdiet sodales eros. Phasellus maximus tristique erat, a viverra metus imperdiet ac. Vivamus faucibus ante id porttitor volutpat.

Vestibulum ullamcorper eleifend velit, id tempus nisl varius et. Cras vitae nunc nibh. Nam dictum facilisis dictum. Cras blandit id tellus sed blandit. Sed porta eros id mi venenatis facilisis. Nullam sed felis sed tellus dictum lobortis bibendum ac libero. Aenean et mollis ipsum. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent faucibus faucibus fermentum. Vestibulum id risus placerat, rhoncus arcu eu, tincidunt lectus. Sed interdum venenatis fermentum. Mauris ultricies nisl diam, sit amet fermentum dui blandit sed.

Fusce tempor semper tellus vitae facilisis. Nam suscipit sapien vel nulla porta aliquet. Vestibulum fringilla condimentum dictum. Nullam pharetra dolor non mauris fermentum, eget elementum nisi dictum. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Sed eu tristique erat, vitae lacinia felis. Sed tincidunt magna quis luctus pulvinar. Donec commodo, velit iaculis ullamcorper posuere, elit nunc commodo felis, at sagittis ex sapien vitae purus.

Donec ac ligula in dui laoreet aliquet. Etiam lobortis arcu lectus, facilisis bibendum turpis feugiat sit amet. In in ex non arcu egestas luctus. Aenean eget diam vel ex auctor pretium. Vivamus elementum ac mauris eu interdum. Curabitur maximus elit sed egestas dictum. Phasellus mattis velit vel enim bibendum blandit. Nullam hendrerit est eros. Vivamus eu tincidunt quam. Sed vel efficitur erat. Etiam sit amet diam a arcu malesuada vulputate. Ut mattis justo id velit lacinia suscipit. Sed sit amet massa ante. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean fermentum id mi id bibendum. Vestibulum metus velit, consequat ac imperdiet mollis, ultrices et nunc.

In hac habitasse platea dictumst. Cras sit amet vestibulum turpis, quis posuere nunc. Proin cursus sapien laoreet felis vehicula, et condimentum diam ultrices. Vivamus a accumsan ligula. Aliquam tempor sollicitudin quam, eu auctor odio dapibus sit amet. Proin interdum rhoncus lacus sit amet feugiat. Maecenas non dui et sem consectetur rutrum quis sed nunc. Maecenas lobortis neque lorem, sed tincidunt justo pharetra non.

Nam vel mauris sit amet ex viverra accumsan. Proin vel rutrum leo. Mauris mi nisl, mattis eget ornare ut, congue vitae arcu. Nam consequat mi sed fringilla dictum. Vivamus sodales, enim sed condimentum viverra, magna velit commodo tortor, quis dictum lacus mi eget lectus. Vestibulum dignissim dui magna, sit amet tempor est semper vel. Aliquam ornare nunc eu porttitor mollis. Maecenas nec augue nec nisl luctus eleifend sed et ligula.

Morbi sollicitudin nisl mi, sed bibendum enim efficitur sit amet. Ut luctus magna leo, in semper mi imperdiet molestie. Cras molestie augue iaculis sem pharetra, quis congue nisi volutpat. Maecenas sodales interdum justo, a dictum mauris placerat id. Suspendisse eleifend dui vitae tortor pellentesque, eget condimentum nisi euismod. Fusce lobortis, sem et lacinia iaculis, nisl lectus eleifend leo, eget consectetur purus nulla in diam. Maecenas ultricies dignissim mauris eu faucibus. Nulla facilisi. Nunc lobortis justo nec egestas blandit.

Praesent mattis maximus nibh sed ultricies. Morbi venenatis nunc id sollicitudin blandit. Integer commodo hendrerit lorem, id mollis velit luctus ut. Phasellus posuere mi eu vulputate gravida. Nullam euismod, diam at feugiat dictum, nunc felis gravida ante, accumsan elementum purus sem ut lectus. Nam lacinia odio tristique, maximus tellus ac, finibus nibh. Maecenas venenatis ipsum sed orci euismod tincidunt. Nulla convallis mauris non lectus ultricies semper.

Quisque sed sodales velit. Nunc gravida commodo scelerisque. Nam ultricies posuere neque in tincidunt. Proin a massa id metus egestas vehicula eget quis eros. Interdum et malesuada fames ac ante ipsum primis in faucibus. Phasellus sit amet scelerisque enim, in posuere quam. Sed orci ante, tempus eget justo et, vestibulum faucibus eros. Curabitur quam sapien, suscipit quis aliquet nec, aliquet vel ipsum. Cras elit purus, sodales non nulla eu, cursus tristique neque. Phasellus pulvinar pellentesque diam eu pulvinar. In faucibus tincidunt mi.

Nunc tincidunt diam non nulla auctor iaculis. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec a placerat nisl. Praesent dictum, velit eget fermentum aliquam, sapien eros ultrices enim, eu pharetra mauris dui a risus. In hac habitasse platea dictumst. Nullam ut elit non nisl scelerisque fringilla quis id velit. Phasellus eu dui sodales, mollis diam quis, commodo quam. Nulla cursus dapibus odio ut placerat. Curabitur id rutrum lorem. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Vivamus sed cursus risus, nec eleifend orci. Nulla auctor, enim non auctor viverra, nisl mi viverra sem, eu molestie leo nisi non nibh. In dolor neque, accumsan ut felis et, mattis cursus purus.

Suspendisse nisl dolor, lacinia at vulputate et, accumsan sed magna. Aenean blandit massa at eros lobortis luctus. Cras ac arcu vitae turpis consequat lacinia. Pellentesque pellentesque nisi in turpis mattis mattis. Aliquam erat volutpat. Suspendisse accumsan est dapibus tellus finibus venenatis. Quisque blandit tellus id rhoncus mattis. Suspendisse potenti. Aliquam ultricies, libero vel condimentum volutpat, nunc velit posuere urna, sed hendrerit quam sapien vitae sapien.

Nulla euismod faucibus ipsum in euismod. Fusce sed maximus nisl. Ut sit amet risus convallis, condimentum elit ac, viverra metus. Vivamus dignissim ligula eu tellus scelerisque, non vestibulum orci malesuada. Aliquam congue at felis non rhoncus. Nullam dapibus eros risus, in tristique ipsum commodo ac. Sed porttitor eros vel libero placerat, et molestie enim scelerisque. Praesent eu sapien dictum, laoreet libero suscipit, gravida sem. Donec auctor dolor a orci rhoncus porttitor. Maecenas nec volutpat ipsum. Donec commodo eu metus quis finibus. Quisque mi quam, dictum non ullamcorper id, placerat sit amet libero. Praesent interdum aliquam ornare. Curabitur blandit lorem arcu, sit amet semper elit elementum eu. Proin sit amet orci eget mi viverra semper.

Nullam felis elit, consectetur sed volutpat nec, tincidunt et erat. Sed quis facilisis neque. Morbi hendrerit nisi dolor, quis egestas odio iaculis sed. Sed ultricies magna non nisi aliquet, id facilisis dui convallis. Suspendisse commodo augue ac lobortis facilisis. Ut magna dolor, condimentum vitae feugiat in, ullamcorper nec urna. Donec cursus ultrices nibh vel dignissim. Proin enim purus, dapibus vitae tempor et, tristique ac libero. Nullam mi tellus, consequat nec sem sed, pellentesque feugiat dolor. Cras luctus vestibulum neque et rhoncus. Pellentesque laoreet quam id porta fringilla. Proin commodo egestas turpis id venenatis. Pellentesque dictum massa eu gravida pulvinar. Vivamus eget finibus ante. Donec eu urna et urna volutpat feugiat. Phasellus ex est, ultricies ut risus ac, finibus posuere dui.

Praesent nulla lectus, tincidunt non pharetra et, ultrices nec enim. Aenean ultricies rhoncus nisl, fringilla accumsan tortor pretium ac. Suspendisse efficitur tempor suscipit. Cras finibus eros vel eros semper condimentum. Aliquam quis odio quis enim aliquam fringilla sed vitae nisi. Ut dignissim erat odio, non eleifend orci maximus tincidunt. Vestibulum tincidunt, ligula vitae cursus tincidunt, nulla orci accumsan orci, eu tincidunt arcu orci vitae odio. Nulla facilisi. Donec at lectus eget metus vulputate cursus non venenatis quam. Proin nisi nisi, sodales id molestie quis, fringilla ut sapien. Donec maximus tempor libero, in pretium urna luctus suscipit. Morbi egestas ac ante ut feugiat. Donec tincidunt maximus neque vitae porta.

Fusce massa est, tristique nec lobortis nec, pharetra id augue. Ut finibus quam et ex varius imperdiet. In sed lectus sit amet nisl vehicula sodales id ac ex. Vivamus sit amet pulvinar dolor. Proin elementum dolor id sollicitudin pulvinar. Curabitur quis nunc elit. Nulla ac consequat lectus, non varius diam. In faucibus id lorem non feugiat. In viverra massa at turpis tempor, sed consectetur velit malesuada.

Integer vestibulum, metus ut tempor ultrices, neque quam gravida eros, at eleifend orci quam vel leo. Pellentesque eleifend porta libero, suscipit aliquam arcu volutpat non. Fusce id consectetur nunc. Nullam eleifend semper lorem, vitae blandit libero tempus ac. Quisque elementum ligula pellentesque feugiat dapibus. Nullam luctus in mi eget rutrum. Sed faucibus pulvinar libero at congue. Sed vestibulum orci a diam tempus venenatis sed in quam. Pellentesque sit amet convallis lorem, a ullamcorper tellus. Nunc rutrum dui ac odio accumsan viverra. Duis lacinia risus in orci accumsan sagittis. Fusce at aliquam lacus, eu feugiat ex. Mauris neque diam, malesuada eu dui sed, tincidunt euismod mi. Fusce non elit varius, ornare nunc sit amet, tempus urna. Mauris erat turpis, tempus vel nulla sit amet, malesuada lacinia est.

Pellentesque et eleifend nisi, quis ullamcorper ligula. Duis eu urna quis augue suscipit blandit. Nulla sodales vel dui nec condimentum. Donec gravida, turpis sed ornare maximus, erat risus vestibulum ex, vitae volutpat quam sapien eget mauris. Integer efficitur bibendum metus nec suscipit. Suspendisse ac eros a dolor aliquet ultrices eu auctor ipsum. Nulla feugiat tristique urna ut aliquet. Vestibulum mi purus, dignissim sit amet porttitor eu, efficitur eget elit. Duis sit amet est eget lorem tincidunt dictum.

Proin congue finibus nunc, at fermentum dui. Nullam auctor eleifend elit quis malesuada. Ut sit amet leo rhoncus, semper arcu ut, ornare ante. Fusce feugiat non mauris sit amet pulvinar. Pellentesque sagittis fermentum massa, a varius quam feugiat eget. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Aenean eget urna nec odio scelerisque lacinia sit amet a ante. Quisque et posuere mi. Aenean congue posuere imperdiet. Nullam et mauris suscipit, sollicitudin lectus vitae, eleifend nisl. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Vestibulum imperdiet laoreet neque. Ut quis magna sodales, molestie arcu at, dictum lectus.

Praesent in commodo lorem. Curabitur at velit nisi. Donec scelerisque quam mi, sed porttitor ex varius malesuada. Curabitur a nibh nec justo dictum faucibus non volutpat diam. Ut ultrices luctus leo, vitae pretium tellus posuere eget. Nulla ligula magna, accumsan eget quam vitae, tempus dictum ante. Morbi efficitur ut augue eget efficitur. Nulla consequat, lacus vitae bibendum efficitur, ante ex porttitor odio, ut porta lectus nisi quis ipsum. Sed vulputate in ex eget rutrum.

Aliquam erat volutpat. Praesent quis nisi luctus, accumsan dolor et, hendrerit orci. Phasellus tempor ante quis ex auctor interdum vestibulum sed tortor. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Suspendisse quam justo, dapibus eget ex eget, molestie lacinia ipsum. Etiam risus felis, rutrum id tincidunt in, ullamcorper ut diam. Mauris et accumsan dui. Vivamus eleifend euismod luctus.

Fusce luctus lorem id metus hendrerit semper. Sed accumsan, urna non vehicula accumsan, mi metus imperdiet dolor, a porttitor nulla dolor id sem. Fusce tellus massa, viverra quis fermentum mattis, finibus lacinia lorem. Ut vulputate augue ut lectus porta semper. Donec quis pharetra ligula. Etiam ut ante ligula. Fusce lobortis tempus urna cursus faucibus. Mauris pellentesque dolor sed condimentum rhoncus. Fusce fringilla aliquet turpis, mattis dignissim elit sollicitudin in. Proin placerat viverra finibus.

Praesent efficitur sed arcu ut venenatis. Praesent facilisis consectetur eleifend. Vestibulum molestie dui vitae rutrum convallis. Ut interdum, diam eget eleifend hendrerit, turpis quam molestie turpis, vel suscipit augue diam a elit. Vivamus vel malesuada sapien. Duis euismod ipsum erat, quis venenatis lorem ultricies in. Nunc at ultricies urna, sed commodo lacus. Aliquam et tincidunt mi. Quisque ut leo semper, vehicula urna non, semper felis. Phasellus interdum aliquet eros eget egestas.

Donec scelerisque sit amet ante accumsan iaculis. Vestibulum ac mi non augue pharetra laoreet non vitae tortor. Sed magna lorem, finibus eget dolor id, blandit porta leo. Etiam semper dolor mi, quis hendrerit nunc cursus nec. Proin interdum quis justo in pretium. Interdum et malesuada fames ac ante ipsum primis in faucibus. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque id faucibus mauris. Suspendisse euismod diam vitae ligula pellentesque aliquet. Praesent ornare dictum neque. Quisque at dolor nec dui lobortis varius. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Ut vitae elit pellentesque, vulputate diam eu, aliquet dui. Curabitur at quam sollicitudin, pretium nibh quis, accumsan est. Phasellus rhoncus cursus eros, eget imperdiet tortor iaculis at.

Praesent semper pretium urna. Fusce mattis pharetra erat, eu ornare arcu dictum eu. Etiam maximus ligula leo, eu ullamcorper massa aliquam tempor. Nunc metus est, lacinia sed aliquet at, hendrerit a tellus. Maecenas cursus id leo in porttitor. Sed vestibulum sodales ex, eget malesuada mauris ornare sit amet. Morbi sed pharetra arcu. Pellentesque gravida convallis libero nec finibus. Sed tellus nisl, fringilla quis mauris ac, tincidunt scelerisque metus. Mauris feugiat tristique odio vel porta. Etiam tincidunt enim quis tellus eleifend, sed pellentesque metus accumsan. Suspendisse consequat commodo sem congue iaculis. Donec egestas eleifend mauris blandit tempor.

Integer ut pellentesque elit. Proin a urna tellus. Aenean efficitur quam ipsum, a laoreet sapien egestas vitae. Fusce commodo consectetur leo, a molestie mi facilisis quis. Etiam varius tempus mauris, nec mattis nisi sagittis eu. Duis faucibus risus sit amet justo venenatis, sed tempor odio aliquam. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Suspendisse et suscipit massa, non hendrerit est. Praesent laoreet quam eget quam eleifend consectetur. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae;

Aliquam et bibendum tortor. Duis condimentum aliquet sollicitudin. Aenean lobortis, nulla vel auctor molestie, magna eros ultrices sapien, in tristique metus risus vel tortor. Nullam ac pellentesque ipsum. Maecenas vitae justo vitae ante fermentum vulputate eget vel augue. Cras laoreet dui enim, a hendrerit purus interdum vel. Pellentesque eleifend ut urna non rhoncus. Sed nec tortor scelerisque, vestibulum est eget, tincidunt dui. Praesent maximus commodo ante, in fermentum lectus dignissim id.

Ut sit amet facilisis arcu. Integer viverra dui id lorem tempor, in pharetra elit dapibus. Pellentesque volutpat suscipit velit sit amet maximus. Curabitur nulla eros, finibus vitae semper sed, commodo ac massa. Aliquam erat volutpat. Proin tempus, turpis suscipit volutpat imperdiet, velit tortor luctus magna, eu fringilla felis enim nec ante. Nunc euismod odio sed hendrerit consectetur. Nam finibus justo sit amet leo rutrum consectetur. Phasellus quis neque gravida, laoreet metus hendrerit, vehicula elit. Morbi pulvinar est vel neque facilisis porta. Nullam ut faucibus neque, sed ornare sem. Duis pulvinar eget nisi et hendrerit. Donec in ante pellentesque metus porta viverra ac non orci. Sed accumsan metus nec ipsum blandit, non commodo urna lobortis. Nunc in est congue, viverra urna non, porttitor est. Curabitur ultrices eros ut quam finibus lobortis.

Mauris tempus consequat urna, ut rutrum ante placerat in. Donec quis pulvinar magna. Duis malesuada condimentum quam vel feugiat. Sed eget magna eget libero tempor ornare. Maecenas nec augue eros. Donec efficitur vestibulum augue in vestibulum. Sed velit ex, viverra in felis a, pharetra mattis eros. Cras ut mi ante.

Vivamus id semper elit, a eleifend mauris. Phasellus venenatis pulvinar massa aliquet ultrices. Maecenas sagittis arcu augue, vel vulputate nibh dignissim sed. Mauris scelerisque luctus erat, eget vestibulum justo. Curabitur id consectetur ex. Quisque a ipsum mattis massa efficitur facilisis. Vestibulum eu augue erat. Praesent in consectetur lorem. Sed placerat semper nibh, vel imperdiet justo. Proin at velit lacus.

Nunc ut enim ac tellus aliquam feugiat malesuada nec ligula. Suspendisse lectus quam, pulvinar vitae mi sit amet, rhoncus dapibus turpis. Fusce laoreet rhoncus orci ut iaculis. Maecenas euismod euismod ipsum, eget sagittis sapien lacinia nec. Vestibulum efficitur lacus et tincidunt elementum. Donec mattis sapien nec magna lobortis, ut pretium odio tempus. Suspendisse porttitor orci sit amet mauris ullamcorper molestie. Proin vitae dignissim purus, a pretium metus. Nam rutrum felis non justo faucibus, id tincidunt sem vestibulum. Fusce auctor, felis et efficitur volutpat, mi felis bibendum ante, eu tincidunt magna nibh non turpis.

Sed accumsan felis diam, non malesuada sem molestie ut. In at feugiat urna, at volutpat justo. Nunc auctor vel diam sed varius. Mauris rutrum ullamcorper dignissim. Donec et fermentum enim. Morbi quis aliquet quam. Quisque semper justo a lectus lacinia condimentum. Pellentesque fringilla vulputate rutrum. Suspendisse potenti. Nullam id gravida tellus.

Nam molestie sollicitudin rhoncus. Interdum et malesuada fames ac ante ipsum primis in faucibus. Morbi posuere metus at metus auctor, ultricies fringilla purus tincidunt. Suspendisse volutpat at dui in aliquet. Etiam ullamcorper nibh enim, quis sodales nisl suscipit sed. Duis vel neque at libero mattis eleifend. Vivamus vehicula, purus eu sollicitudin rhoncus, velit justo commodo nisl, eu pharetra est lectus sit amet tellus. Nulla egestas imperdiet arcu, non iaculis dui pharetra eu. Sed pretium nulla diam, ut hendrerit orci consequat ut. Pellentesque vel fermentum nulla, non ornare nunc.

Aliquam non quam aliquet, cursus eros eget, porta ipsum. Sed fringilla est ac eros pellentesque, a lobortis lacus tincidunt. Proin tincidunt sollicitudin sagittis. Interdum et malesuada fames ac ante ipsum primis in faucibus. Aliquam at magna rhoncus, aliquet ex sit amet, tincidunt urna. Suspendisse efficitur dolor quis dui consectetur ultricies. Donec nibh nulla, eleifend id semper ut, varius in est. Nulla blandit et nibh ut imperdiet.

Donec vitae consectetur odio. Vestibulum vestibulum mauris a pharetra laoreet. Integer pharetra congue porta. Suspendisse rutrum sagittis massa, feugiat bibendum turpis volutpat nec. In neque ipsum, vulputate nec leo non, blandit commodo lacus. Pellentesque pharetra ultrices placerat. Suspendisse a lacinia mi, id congue augue. In tristique felis et ipsum dapibus molestie. Phasellus et metus a velit gravida egestas. Nulla tristique consectetur ipsum at consectetur. Aenean tempus ipsum mauris, a lobortis magna congue sed. Integer risus eros, facilisis at ante non, scelerisque faucibus justo. In molestie est efficitur elementum tristique. In accumsan pretium mi, ac consectetur neque semper scelerisque. Aenean eleifend, urna nec aliquet tempor, quam est suscipit ante, vitae congue leo tellus at eros.

Suspendisse et mauris at nibh suscipit condimentum. Duis eu turpis in erat fermentum ullamcorper vitae vel sapien. Nulla volutpat interdum posuere. Aenean eleifend nec odio quis hendrerit. In eu venenatis nisi, quis condimentum tortor. Vestibulum non est eu nisi efficitur consectetur ut et velit. Suspendisse posuere nunc in turpis cursus facilisis.

Fusce ut placerat nulla. Aenean placerat vel lectus quis semper. Quisque eget semper mi. Vivamus iaculis, nisl sed hendrerit posuere, felis velit elementum nunc, in mollis erat erat eget sem. Donec ultrices, felis rutrum consectetur vehicula, diam lorem cursus metus, ac rhoncus felis ex a mauris. Nulla ullamcorper magna ac turpis maximus, quis blandit arcu dapibus. Sed aliquam, neque vel placerat congue, massa sem congue nisi, vitae pulvinar mi lacus in enim.

In eget ex varius, ornare diam ac, tempus magna. Praesent tincidunt, leo vitae feugiat lobortis, nisl elit luctus leo, vitae sollicitudin nibh nisl ut lacus. Suspendisse nec risus eu nibh molestie efficitur. Duis at quam eros. Nullam ut ultrices nunc. Phasellus id orci purus. Sed tristique, urna sed pharetra pellentesque, lorem tortor luctus nisl, at vestibulum nisl massa dapibus sem. Fusce vel venenatis arcu. Praesent sit amet molestie ligula, eu fermentum leo. Integer libero nulla, posuere et sagittis ut, aliquam ac eros. Ut sodales velit vestibulum urna aliquet scelerisque a eu ipsum. Sed imperdiet lorem ac mollis lobortis. In non dui urna. Nulla sollicitudin, sapien vel consectetur venenatis, neque erat euismod nibh, vitae tempus justo nulla nec nibh. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Integer sit amet nunc molestie ex fringilla tincidunt.

Curabitur non felis nec urna tristique pretium at vel justo. Etiam tempor eros in leo luctus gravida. Cras at sagittis tellus. Aliquam molestie, nibh eget mollis varius, ligula metus varius nibh, eu gravida mauris lectus ac orci. Vivamus non eros nulla. Suspendisse aliquam nisi quis commodo rutrum. Sed posuere nulla lacus, a vehicula ante rutrum et. Fusce in consequat elit, et molestie felis. Maecenas lectus ipsum, pellentesque vitae augue a, consectetur aliquet dolor. Proin luctus vitae erat ac interdum. Quisque sed pharetra ex. In at vulputate urna, at rhoncus ligula. In hac habitasse platea dictumst. Maecenas elit orci, hendrerit eget quam sit amet, posuere lobortis lorem. Nullam lobortis sodales feugiat. In fringilla et neque vitae scelerisque.

Phasellus quis posuere neque. Sed vehicula velit ac justo semper rutrum. Nulla vel dolor in quam viverra congue et vel ligula. Quisque et rutrum nulla. Maecenas id turpis a neque gravida lacinia. Curabitur condimentum, eros quis tincidunt porta, massa magna fringilla nunc, iaculis laoreet ipsum odio vitae nisl. Cras pharetra congue mollis.

Phasellus id libero in lectus tristique iaculis nec in libero. Pellentesque nibh magna, rutrum et fermentum id, dignissim a tortor. Integer lacus lacus, volutpat non fermentum vitae, gravida in sapien. Sed porta quam in urna malesuada fermentum. In hac habitasse platea dictumst. Aliquam fermentum gravida leo at lobortis. Vestibulum quis nulla rutrum, consequat sem vitae, ornare enim. Nullam sed neque quis leo ullamcorper vulputate. Suspendisse non lacus velit. Ut eget orci nec urna pellentesque pretium vitae sed eros. Nulla nulla nisl, vulputate molestie nunc eu, tempor rhoncus nisi. Morbi ut mauris vitae odio maximus efficitur.

Suspendisse potenti. In interdum metus sed dui faucibus, ullamcorper feugiat turpis laoreet. Etiam nulla leo, luctus vitae dui elementum, molestie feugiat est. Pellentesque volutpat metus vel enim euismod, quis tincidunt erat posuere. Nulla facilisi. Quisque sollicitudin turpis quis dui placerat mattis. Praesent porta rutrum dui ut sollicitudin. Integer et est pulvinar, faucibus lectus a, blandit ipsum. Suspendisse accumsan lobortis congue.

Sed maximus arcu ut semper varius. Integer massa metus, pulvinar sed dui ut, consectetur congue lacus. Praesent a tortor id nulla blandit elementum ut eget justo. Vivamus fermentum et nisi eu rhoncus. Vestibulum malesuada justo purus, nec eleifend arcu consequat id. Sed sit amet venenatis lorem, sit amet porttitor eros. Proin ornare tincidunt nunc. Donec ac feugiat tortor. Aliquam a mi nec purus aliquet tempor. Nullam tristique tellus risus, ac rhoncus orci lacinia sit amet.

Aenean eget consectetur lacus, non pellentesque leo. Suspendisse non ante semper, mattis felis et, ornare nibh. Morbi a dignissim est, at malesuada neque. Sed elementum purus in lectus malesuada blandit. Praesent sollicitudin augue leo, sed scelerisque turpis maximus at. Nam in efficitur metus. Pellentesque ultricies, erat sit amet mattis blandit, ipsum ligula lobortis purus, in varius magna felis at libero. Pellentesque aliquet viverra felis et tempor. Nam eget fermentum arcu. Vestibulum pharetra imperdiet justo et scelerisque. Praesent malesuada velit id est lobortis posuere. Vivamus eget justo id ipsum auctor molestie quis eget metus. Ut a viverra ipsum. Nulla consectetur venenatis augue, eu faucibus urna venenatis sit amet. Phasellus porttitor, elit sed dapibus euismod, justo enim vehicula felis, id hendrerit nunc felis id nibh.

Donec feugiat libero vitae ipsum ultricies vestibulum. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Phasellus fermentum nec nisl in vulputate. Donec posuere mauris quis lorem eleifend, a lacinia erat tincidunt. In hac habitasse platea dictumst. Aenean lacus dui, vulputate vitae interdum at, porta ut diam. Integer non massa sed leo imperdiet iaculis ut sed lectus. Cras ultricies feugiat consectetur. Nullam commodo magna rhoncus sollicitudin rutrum. Donec tristique nunc sed odio bibendum, eget luctus dolor gravida. Suspendisse potenti.

Proin mattis nisl erat, id lobortis diam luctus nec. Nullam et elit ultrices, ultricies nunc cursus, mollis augue. Nam felis elit, maximus non tellus eget, volutpat molestie erat. Pellentesque luctus fermentum magna, id commodo turpis blandit sit amet. Nullam at nulla augue. Pellentesque non facilisis neque. Donec non purus diam. Pellentesque pretium sapien a nunc tincidunt, in volutpat eros facilisis. Proin porta mauris eget enim placerat, sed laoreet nisi volutpat. Fusce consequat tincidunt lectus, vel dictum ligula consectetur et. Mauris rutrum vulputate blandit. Ut sodales non quam ut semper. Quisque neque purus, sagittis ut fringilla id, ultricies at augue. Duis ullamcorper libero nec maximus ullamcorper. Mauris aliquam consectetur ante, vitae bibendum odio hendrerit non. Maecenas efficitur pellentesque lectus, nec tempus odio finibus eget.

Ut diam neque, rhoncus at bibendum id, pulvinar non dui. Fusce at lectus non velit euismod posuere. Sed a efficitur arcu. Donec mi felis, sollicitudin in nisi eget, eleifend aliquet metus. Phasellus ut sem facilisis, posuere lacus a, facilisis velit. Nam vulputate nisi nec urna dignissim ultricies. Nullam pulvinar nunc sed porttitor maximus. Cras tincidunt auctor odio, a malesuada dui ultrices eu. Curabitur dapibus luctus malesuada.

Aliquam non metus non sem fringilla tempor. Praesent at velit a mauris cursus tristique quis non erat. Cras id elit laoreet, faucibus orci eu, tempor dolor. Integer eget aliquam ligula. Vivamus commodo eleifend risus non varius. Aliquam sed ornare est, in feugiat turpis. Nam egestas ultrices facilisis. Curabitur feugiat risus nibh, eget efficitur libero aliquam eget. Aenean nisl ex, efficitur eget eros eu, tincidunt consectetur metus. Nulla facilisi. Praesent ac viverra nisi. Pellentesque lacinia nulla sed lacinia porta. Integer in ante ut orci aliquet dapibus.

Ut ornare sed turpis eu pellentesque. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Vivamus hendrerit tincidunt condimentum. Phasellus quis eros eget ex tristique pellentesque. In hac habitasse platea dictumst. Pellentesque non rhoncus metus, eget iaculis justo. Nunc congue fringilla mattis.

Sed ultrices placerat ipsum vitae semper. Aenean elementum consequat nisl, in pharetra magna rutrum sed. Nunc eget urna semper, gravida erat et, rutrum nibh. Proin nisi augue, efficitur sit amet urna fringilla, faucibus placerat neque. Integer eu neque nec felis consectetur fringilla in non nunc. Quisque id erat vitae nunc egestas posuere in sed orci. Nullam tincidunt sollicitudin nibh a rhoncus.

Vivamus tempus, neque at posuere consequat, lorem eros sollicitudin arcu, eget scelerisque ipsum nunc vel tortor. Donec ultricies purus eros, a aliquam neque hendrerit non. Praesent magna urna, posuere in iaculis sed, fermentum et ante. Maecenas nec pellentesque est. Aliquam vitae nisi a orci rhoncus tincidunt. Ut ac semper urna. Etiam vel dapibus diam. Pellentesque mi mi, aliquet non egestas ac, feugiat ac lectus. Suspendisse gravida molestie urna, at laoreet nunc efficitur et.

In sit amet gravida risus, nec posuere mauris. Curabitur id tempor risus. Nunc enim purus, laoreet vel libero ac, imperdiet lobortis mauris. Sed eget hendrerit diam, imperdiet scelerisque urna. Donec eget nisl in erat rhoncus pulvinar. In eget eleifend eros. Nunc tristique tempus nulla, semper hendrerit ipsum congue eu.

Duis in interdum nunc, eu venenatis lorem. Sed semper scelerisque dui, sed posuere lorem dignissim sit amet. Ut tempor a diam id scelerisque. Quisque suscipit posuere nunc vitae auctor. Morbi dapibus porttitor auctor. Vivamus bibendum pellentesque fermentum. Aliquam vel quam a risus congue consectetur. Donec sed erat metus. Cras interdum faucibus augue. Curabitur ullamcorper risus a velit elementum, blandit egestas nibh malesuada. Quisque erat urna, posuere quis mauris in, bibendum mollis ex. Mauris eget leo vel lorem viverra eleifend a et sapien. Etiam in efficitur purus.

Integer sed nisl sit amet enim viverra pulvinar id vitae dolor. Nulla pharetra nibh est, at dapibus lectus ultricies nec. Ut lacinia gravida dolor nec ornare. Proin quis arcu eget ipsum lacinia scelerisque. Ut dapibus et lacus sit amet mollis. Integer fringilla dapibus tellus at ultricies. Morbi rhoncus nibh at augue vulputate pulvinar. In et elit mollis, varius nibh non, mollis lorem. Aliquam non lacus nulla. Praesent blandit mattis enim, in elementum tellus porta sit amet. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Donec volutpat malesuada elit. Nullam cursus ex ac tempus iaculis. Nam pretium, magna vel pretium sagittis, ex dolor tempor elit, sed elementum leo leo eu mauris.

Vestibulum malesuada tincidunt leo vel lobortis. Maecenas sit amet felis ante. In aliquam orci sem. Phasellus laoreet metus non libero tempus sollicitudin. Mauris id dictum ipsum, nec aliquet erat. Aenean sodales ligula est, id dapibus tortor pharetra eu. Etiam arcu ipsum, dapibus ac arcu at, bibendum finibus libero. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Sed quis nisi congue, porttitor augue quis, volutpat ipsum. Aliquam tincidunt nibh quis ex cursus, ac dignissim nibh imperdiet. Mauris nec mi non leo mollis eleifend eu vel magna.

In dignissim, augue eget interdum mattis, libero enim gravida ante, sit amet convallis risus nibh id mi. Integer tristique auctor nisl nec viverra. Integer bibendum ipsum a fermentum aliquet. Phasellus tincidunt elit metus, ut placerat sapien fermentum et. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Vestibulum at leo eget diam varius convallis. Donec tempor, elit vel malesuada euismod, libero massa feugiat sem, sit amet condimentum nisl libero et mauris. Integer vel mi felis. Fusce in tincidunt mi, non euismod ipsum. Mauris mollis ornare sem vel ultricies. Morbi id finibus metus, eu mattis neque. Pellentesque vel eros at mi dapibus mollis id eget nunc. Vivamus aliquet maximus nisl, id vestibulum arcu volutpat quis. Mauris gravida nisi at diam elementum egestas.

Donec tincidunt justo sed mauris malesuada, scelerisque pellentesque nulla euismod. Aenean egestas diam sed maximus elementum. Nunc pharetra molestie velit, vitae cursus libero varius et. Fusce pellentesque vulputate ultrices. Integer volutpat mauris sed laoreet dignissim. Maecenas et augue tincidunt, vehicula eros at, tempus mi. Nam sagittis, tortor vel consectetur pharetra, velit eros auctor mauris, at pretium augue risus euismod neque. Etiam a facilisis ipsum. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Nulla pretium neque id massa ornare, ut condimentum sem lacinia.

Etiam nec convallis metus. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Donec ut mauris ac dolor auctor pharetra sit amet ac tellus. Pellentesque suscipit turpis et mi ornare ultrices. Etiam dignissim condimentum arcu, sed tristique lectus ultrices at. Vivamus quis sapien at lacus efficitur consectetur a vel nisi. Nam malesuada ipsum sed risus mattis ultrices. Praesent vitae mollis tellus, non fringilla justo. Maecenas egestas placerat odio, at vestibulum urna tincidunt at. Fusce blandit ac libero a scelerisque.

Suspendisse aliquam consectetur sollicitudin. Proin porttitor dui neque, id ultricies lorem pellentesque eu. In a enim ac elit auctor scelerisque. Ut sit amet eros vulputate, consequat purus et, vehicula magna. Praesent dui odio, suscipit non sagittis at, tempor a ex. Ut pretium eleifend facilisis. Integer tristique libero velit, maximus ornare lacus pellentesque non. Vivamus suscipit vel leo eu tempus. In a auctor mauris.

Cras bibendum, nunc congue porttitor luctus, quam dolor iaculis leo, at viverra arcu odio non diam. Fusce sodales porta neque. Nulla euismod lectus at lacus lacinia dignissim. Curabitur vitae augue vehicula, vulputate felis sit amet, ornare augue. Sed ac aliquet nulla, et molestie ante. Proin posuere non quam et finibus. Nam ipsum felis, sodales ac luctus vel, elementum non purus. Aliquam hendrerit diam quis tortor varius egestas. Etiam euismod vel leo vitae hendrerit. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Aliquam sagittis dolor arcu, quis volutpat enim rhoncus id. Fusce interdum, justo sit amet dictum pretium, libero nibh cursus justo, in imperdiet sem mauris in orci. Morbi quis venenatis ligula, ullamcorper tristique odio. Morbi hendrerit id risus sed lacinia. Quisque et dictum mauris, nec euismod justo.

Maecenas vel purus eu lectus vulputate gravida non non nulla. Proin tincidunt libero ac est elementum, id euismod ante dignissim. Aliquam ante ex, auctor sed turpis sit amet, aliquam vestibulum risus. Nulla placerat tempor nisi, sed aliquam purus efficitur eget. Sed elementum laoreet enim, a rutrum ante vulputate ac. Duis blandit ex a metus porttitor facilisis. Maecenas mattis, elit ac pulvinar placerat, sapien purus blandit lacus, nec accumsan ipsum diam in nunc. Suspendisse at enim vitae enim rutrum volutpat blandit non lorem. Praesent congue erat quis lacus auctor, non blandit lectus convallis. Maecenas sit amet orci eu elit porta tincidunt non sit amet massa. Mauris fermentum at turpis at lobortis. Nullam dapibus tempor aliquam. Vestibulum eget interdum nulla. Sed accumsan venenatis vestibulum. Aliquam est ligula, fermentum quis dui id, dignissim auctor elit.

Vestibulum aliquam pharetra mauris vitae rutrum. Sed ut tempus lectus. Duis vestibulum erat orci, pellentesque lobortis diam iaculis eu. Vivamus aliquam porta ante ultricies suscipit. Integer in quam eu elit dignissim auctor sed vel ligula. Etiam rutrum, sem eget posuere feugiat, dui tortor fringilla eros, faucibus eleifend odio eros in nunc. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Aliquam et quam sed elit viverra efficitur. Sed sit amet mauris est. Aliquam pulvinar lorem nunc, ut vehicula ligula mollis ut. Vestibulum imperdiet lacus mattis ex euismod, sit amet tincidunt quam rutrum.

Cras aliquam augue sed nulla tincidunt laoreet. Integer ornare odio augue, nec scelerisque metus cursus at. Nullam non est vel erat tempus vehicula. Nulla a vulputate nibh. Mauris varius cursus sollicitudin. Donec consequat, magna eget accumsan fermentum, libero enim tincidunt felis, vel lacinia purus nunc semper ex. Maecenas laoreet finibus pulvinar. Ut pellentesque suscipit aliquam. Donec lacus nisl, molestie nec suscipit eu, imperdiet eget metus. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Vestibulum vel enim libero. Vestibulum semper libero id imperdiet placerat. Suspendisse porttitor eget tellus ac laoreet.

Mauris ornare metus quis elit consectetur vulputate. Etiam massa orci, interdum et justo ut, condimentum lobortis turpis. Aliquam vel congue libero. Fusce justo urna, congue non tortor a, molestie rhoncus mauris. Ut ut tellus nunc. Aliquam odio ex, lobortis ac blandit volutpat, blandit sed leo. Nunc tincidunt sodales diam sed egestas.

Aliquam placerat nisl sed mauris aliquet ultricies. Nunc maximus diam et commodo scelerisque. Etiam ac felis vulputate, efficitur odio non, pharetra risus. Ut ut metus velit. Ut lacinia eget dolor nec sagittis. Nullam ut tempor metus. Duis non sapien eget quam pharetra ultricies. Etiam tortor nulla, consequat eu venenatis sit amet, euismod id leo. Donec bibendum dui lacus, quis condimentum lacus hendrerit eu. Quisque congue nunc in interdum vulputate. Fusce ac felis finibus, venenatis dui ut, aliquet orci. Nunc tempus aliquam molestie. Mauris nec lorem vel ipsum euismod lobortis. Fusce id dui interdum, finibus quam a, tincidunt nibh. Mauris sit amet risus nec tortor vulputate varius nec eget lacus. Nullam commodo ligula pellentesque tempus rhoncus.

Fusce auctor hendrerit ligula, consequat vestibulum orci viverra eu. Praesent est mauris, mattis ac ligula eu, aliquam congue ipsum. Nullam rutrum ac diam vitae semper. In interdum molestie nisl in euismod. In scelerisque, lorem id aliquam rutrum, risus est consequat est, a finibus metus magna sed risus. Maecenas metus erat, porta convallis leo a, posuere consequat nunc. Donec rutrum lacus risus, nec porttitor dui pretium at. Fusce et sem dictum metus cursus aliquam nec eget enim. Cras congue mollis sapien ullamcorper molestie. Cras sit amet tincidunt odio. Nunc sagittis quis elit in elementum. Nullam vel purus enim. Sed sed malesuada felis, et molestie nibh. Sed rutrum sapien sem, ut convallis odio pulvinar ac. In a neque lectus. Sed iaculis vitae metus et malesuada.

Pellentesque pharetra auctor neque, at convallis dui vulputate sed. Donec lacinia diam ut mauris aliquam blandit. Ut ultrices turpis ligula, eu aliquet massa imperdiet tincidunt. Pellentesque et rutrum ante. Nulla augue nibh, molestie vitae libero ac, mattis rhoncus lectus. Quisque ante ante, vehicula non lacinia a, sollicitudin sed odio. Nulla porta, nibh quis sollicitudin tempor, nisl ex ultrices arcu, at vestibulum massa lectus at orci. Donec ornare augue ac tellus suscipit, vitae congue nunc tempor.

Nullam malesuada sagittis enim eget facilisis. Nam in fermentum arcu. Vestibulum rhoncus dolor id lorem ornare iaculis. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Mauris sed magna sollicitudin, ullamcorper dui sit amet, gravida sem. Maecenas non volutpat est, sit amet mattis libero. Morbi ut libero eget massa congue fermentum vel non purus. Mauris mi felis, rutrum sit amet lorem a, pretium sagittis metus. In et tortor at diam pharetra viverra. Curabitur hendrerit condimentum lorem quis sodales.

Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Maecenas facilisis aliquet velit, quis luctus odio vehicula eu. Maecenas pharetra, erat ut ultricies eleifend, urna lacus vulputate enim, at scelerisque leo augue eu arcu. Aenean aliquet metus eu metus tempor accumsan. Vivamus et maximus risus. Morbi euismod massa blandit laoreet commodo. Sed velit turpis, condimentum id auctor in, hendrerit sit amet dolor. Donec bibendum et urna eu ultrices.

Praesent metus metus, bibendum quis egestas vitae, mattis hendrerit velit. Donec mollis mauris dui, vitae porttitor ipsum ultrices a. Nullam accumsan nulla sem. Maecenas diam orci, eleifend a euismod vitae, accumsan ut arcu. Vivamus massa ipsum, ornare non urna id, luctus mollis orci. Etiam id dolor odio. Nulla ornare pharetra purus ac pharetra. Ut blandit tortor et turpis vehicula, nec porttitor odio mattis. Nam feugiat venenatis lectus, a dignissim ex mattis porttitor. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Maecenas lorem nisi, imperdiet vel vestibulum eget, consectetur ac est.

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Maecenas quis nunc finibus, pellentesque diam non, tempor quam. Nullam semper tempor aliquet. In accumsan ornare metus sed volutpat. Integer eget sagittis lectus, sed feugiat tortor. Suspendisse eleifend metus sit amet nunc fermentum molestie. Fusce varius ante a augue interdum dictum. Etiam mattis dictum finibus. Donec sed vulputate est, in posuere neque. Sed at ex vitae metus hendrerit efficitur vitae ut diam.

Maecenas sit amet augue diam. Fusce interdum posuere elit nec sodales. Nulla finibus nunc neque. In est ex, luctus id ligula ut, efficitur volutpat risus. Mauris blandit, quam non dapibus sodales, erat risus sodales nunc, eu semper nisl risus eu nunc. Vivamus et justo ligula. Nam condimentum, purus eget dapibus hendrerit, nibh nibh aliquet ex, sit amet tempor lectus sem quis lectus. Nullam sit amet quam tempor, ullamcorper odio a, accumsan ex. Curabitur et sem varius, porta purus ut, egestas nibh.

Phasellus tempor velit elit, at tempor odio pharetra vel. Donec egestas mattis justo. Cras rhoncus consectetur risus sed volutpat. Sed et neque libero. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Duis molestie velit ac lacus tincidunt, in consectetur massa semper. Aliquam faucibus, quam sed feugiat sodales, neque enim commodo nibh, vitae feugiat ligula orci vitae nisi. Proin nec viverra enim. Nullam vulputate malesuada diam quis hendrerit. Ut dolor nunc, vestibulum in consequat eu, hendrerit nec velit. Aenean commodo dolor ut orci lobortis, in vulputate dui accumsan. Nullam sagittis, arcu id posuere molestie, augue elit ornare metus, nec imperdiet lorem ex in erat. Nam pellentesque ipsum sapien, vel cursus nibh ullamcorper vitae. Cras sed tristique nibh, ut facilisis ante. Nulla tempor at sapien nec molestie. Suspendisse potenti.

In placerat augue elit, gravida luctus odio luctus vel. Duis eget aliquet tellus. Vestibulum vehicula molestie leo, vel semper felis scelerisque vel. Integer vestibulum congue dapibus. Maecenas gravida diam mauris, nec blandit lorem tempor nec. Praesent nec blandit mi. Morbi fermentum egestas mi in pellentesque. Fusce in efficitur tortor, vitae tristique urna. Phasellus mollis augue sed lacus euismod luctus. Mauris interdum at sem eget aliquam. Praesent luctus massa pharetra, venenatis lectus feugiat, viverra odio. Integer egestas mi at nulla viverra tincidunt. Nulla facilisi.

Nulla facilisi. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. In vel malesuada odio, ut congue metus. Morbi turpis odio, sollicitudin nec elementum eget, hendrerit in erat. Pellentesque luctus nunc vitae lectus pharetra, non tincidunt lacus volutpat. Praesent convallis tempor eros, eu semper risus auctor vitae. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Proin in semper ante, quis euismod dui. Curabitur quis tincidunt mauris. Etiam a semper sapien. Vivamus pretium, felis sit amet pretium luctus, nunc dui pulvinar libero, sit amet dapibus nibh felis eget quam. Pellentesque ligula ligula, volutpat vitae justo ut, aliquam accumsan metus. `,
	}

	for i := range expected {
		//create partitions
		partitions, _ := Partition([]byte(expected[i]), []byte{0x05})
		//strip front matter from partitions
		for j := range partitions {
			strippedPartition, err := ValidatePartition(partitions[j])
			if err != nil {
				t.Errorf("Didn't validate a valid partition: %v, %v", j, err.Error())
			}
			partitions[j] = strippedPartition.Body
		}
		// assemble stripped partitionsj
		actual := Assemble(partitions)

		if string(actual) != expected[i] {
			t.Errorf("Actual (length %v): %v", len(string(actual)), string(actual))
			t.Errorf("Expected (length %v): %v", len(expected[i]), expected[i])
		}
	}
}

// We need to be sure that these invalid payloads get rejected for collation
// without crashing the client with an out of bounds array access.
func TestValidatePartition(t *testing.T) {
	invalidPayloads := [][]byte{
		// empty
		{},
		// ID only
		{0x05},
		// ID only, and is too long to have been generated according to our
		// expectations
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0f},
		// no ID, only index and max index
		{0x3f, 0xff},
		// ID without an ending byte
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		// ID and index info without a payload
		{0x00, 0x00, 0x00},
	}

	// these contain a variable-length ID,
	// an index that's less than or equal to the max index,
	// a max index that's greater than or equal to the index,
	// and a message after the front matter
	validPayloads := [][]byte{
		// Message 2 of 2 with id 0. Note that we're appending a payload to this
		{0x00, 0x01, 0x01},
		// Message 1 of 1 with id 0. Note that we're appending a payload to this
		{0x00, 0x00, 0x00},
		// Note that in some cases, the system validates something that's
		// readable. In this case, the first three letters will be consumed.
		[]byte("telecommunication is neat"),
		// This test case is one that should be valid but failed to validate
		// in the integration test during development after passing through the
		// whole system.
		// Putting it here for posterity.
		{0, 0, 0, 1, 10, 8, 8, 216, 153, 249, 217, 5, 24, 1, 18, 0, 26, 8,
			72, 101, 108, 108, 111, 44, 32, 50},
	}
	expectedIDs := [][]byte{{0x00}, {0x00}, {'t'}, {0}}
	expectedIndexes := []byte{0x01, 0x00, 'e', 0}
	expectedMaxIndexes := []byte{0x01, 0x00, 'l', 0}
	expectedBodies := [][]byte{
		[]byte("apples and grapes"),
		[]byte("apples and grapes"),
		[]byte("ecommunication is neat"),
		{1, 10, 8, 8, 216, 153, 249, 217, 5, 24, 1, 18, 0, 26, 8,
			72, 101, 108, 108, 111, 44, 32, 50},
	}

	// make first two payloads valid by adding a payload to them
	for i := 0; i < 2; i++ {
		validPayloads[i] = append(validPayloads[i], []byte("apples and grapes")...)
	}

	for i := range invalidPayloads {
		_, err := ValidatePartition(invalidPayloads[i])
		if err == nil {
			t.Errorf("Payload %v was incorrectly validated.", i)
		}
	}

	for i := range validPayloads {
		result, err := ValidatePartition(validPayloads[i])
		if err != nil {
			t.Errorf("Payload %v was incorrectly invalidated: %v", i, err.Error())
		}
		if !bytes.Equal(result.ID, expectedIDs[i]) {
			t.Errorf("Payload %v's ID was parsed incorrectly. Got %v, "+
				"expected %v", i, result.ID, expectedIDs[i])
		}
		if result.Index != expectedIndexes[i] {
			t.Errorf("Payload %v's index was parsed incorrectly. Got %v, "+
				"expected %v", i, result.Index, expectedIndexes[i])
		}
		if result.MaxIndex != expectedMaxIndexes[i] {
			t.Errorf("Payload %v's max index was parsed incorrectly. Got %v, "+
				"expected %v", i, result.MaxIndex, expectedMaxIndexes[i])
		}
		if !bytes.Equal(result.Body, expectedBodies[i]) {
			t.Errorf("Payload %v's body was parsed incorrectly. Got %v, "+
				"expected %v", i, result.Body, expectedBodies[i])
		}
	}
}
