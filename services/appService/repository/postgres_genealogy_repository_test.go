package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Real Postgres, like every other *_repository_test.go: a mocked database
// would not prove the SQL, the CHECKs or the locking.

type genRepos struct {
	db          database.Database
	genes       domain.GenealogyRepository
	members     domain.GenealogyMemberRepository
	viewers     domain.ViewerRepository
	invitations domain.InvitationRepository
	phones      domain.UserPhoneLookup
}

func newGenRepos(t *testing.T) *genRepos {
	t.Helper()
	db := connectTestDB(t)
	return &genRepos{
		db:          db,
		genes:       NewGenealogyRepository(db),
		members:     NewGenealogyMemberRepository(db),
		viewers:     NewGenealogyViewerRepository(db),
		invitations: NewGenealogyInvitationRepository(db),
		phones:      NewUserPhoneLookup(db),
	}
}

func genLimits() domain.MemberLimits {
	return domain.MemberLimits{MaxMembers: 5000, MaxGeneration: 60, MaxSpouses: 5}
}

func person(name string, g domain.Gender, kind domain.MemberKind) domain.GenealogyMember {
	return domain.GenealogyMember{FullName: name, Gender: g, Kind: kind, IsAlive: true}
}

// createTree makes a tree owned by owner with a founder and returns its id and
// the founder's member id.
func (r *genRepos) createTree(t *testing.T, owner int64, name string) (gid, founderID int64) {
	t.Helper()
	founder := person("Tổ", domain.GenderMale, domain.MemberKindBlood)
	founder.Generation = 1
	gid, err := r.genes.Create(context.Background(), owner, domain.CreateGenealogyParams{
		Name: name, AncestralHome: "Nam Định", Founder: founder,
	}, 10)
	require.NoError(t, err)
	all, err := r.members.List(context.Background(), gid)
	require.NoError(t, err)
	require.Len(t, all, 1)
	return gid, all[0].ID
}

func (r *genRepos) addChild(t *testing.T, gid, parentID int64, name string, g domain.Gender) *domain.GenealogyMember {
	t.Helper()
	m := person(name, g, domain.MemberKindBlood)
	got, err := r.members.Add(context.Background(), gid, domain.RelationChild, parentID, &m, genLimits())
	require.NoError(t, err)
	return got
}

func (r *genRepos) addSpouse(t *testing.T, gid, ofID int64, name string, g domain.Gender) *domain.GenealogyMember {
	t.Helper()
	m := person(name, g, domain.MemberKindInLaw)
	got, err := r.members.Add(context.Background(), gid, domain.RelationSpouse, ofID, &m, genLimits())
	require.NoError(t, err)
	return got
}

func TestGenealogyRepository_CreateAndFind(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")

	founder := person("Nguyễn Văn A", domain.GenderMale, domain.MemberKindBlood)
	founder.Generation = 1
	founder.IsClanHead = true
	wife := person("Trần Thị B", domain.GenderFemale, domain.MemberKindInLaw)
	wife.Generation = 1

	gid, err := r.genes.Create(ctx, owner, domain.CreateGenealogyParams{
		Name: "Họ Nguyễn", AncestralHome: "Nam Định", Founder: founder, Spouses: []domain.GenealogyMember{wife},
	}, 10)
	require.NoError(t, err)

	d, err := r.genes.FindDetail(ctx, gid)
	require.NoError(t, err)
	assert.Equal(t, "Họ Nguyễn", d.Name)
	assert.Equal(t, owner, d.OwnerUserID)
	assert.Equal(t, "Nguyễn Văn A", d.FounderName)
	assert.Equal(t, 1, d.GenerationsCount)
	assert.Equal(t, 2, d.MembersCount, "founder plus one spouse")
	require.NotNil(t, d.ClanHeadMemberID)
	assert.Equal(t, gid, d.Chronicle.GenealogyID, "an empty chronicle is created with the tree")

	all, err := r.members.List(ctx, gid)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, domain.MemberKindBlood, all[0].Kind)
	require.NotNil(t, all[1].SpouseOfID)
	assert.Equal(t, all[0].ID, *all[1].SpouseOfID)
	assert.Equal(t, 1, all[1].Generation, "an in-law shares the generation of the member it is attached to")
}

func TestGenealogyRepository_CreateLimit(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")

	p := func() domain.CreateGenealogyParams {
		f := person("Tổ", domain.GenderMale, domain.MemberKindBlood)
		f.Generation = 1
		return domain.CreateGenealogyParams{Name: "x", AncestralHome: "y", Founder: f}
	}
	for i := 0; i < 2; i++ {
		_, err := r.genes.Create(ctx, owner, p(), 2)
		require.NoError(t, err)
	}
	_, err := r.genes.Create(ctx, owner, p(), 2)
	assert.ErrorIs(t, err, domain.ErrGenealogyLimitReached)

	// Another owner has their own allowance.
	other := insertUser(t, r.db, "0900000002", "p@example.com")
	_, err = r.genes.Create(ctx, other, p(), 2)
	require.NoError(t, err)
}

func TestGenealogyRepository_CreateConcurrentlyNeverExceedsTheLimit(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")

	const attempts, max = 8, 3
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f := person("Tổ", domain.GenderMale, domain.MemberKindBlood)
			f.Generation = 1
			_, err := r.genes.Create(ctx, owner, domain.CreateGenealogyParams{Name: "x", AncestralHome: "y", Founder: f}, max)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok := 0
	for err := range results {
		if err == nil {
			ok++
		}
	}
	assert.Equal(t, max, ok, "exactly max creations succeed under concurrency")
	list, err := r.genes.ListByUser(ctx, owner)
	require.NoError(t, err)
	assert.Len(t, list, max)
}

func TestGenealogyRepository_CreateIsAtomic(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")

	phone := "+84911111111"
	f := person("Tổ", domain.GenderMale, domain.MemberKindBlood)
	f.Generation = 1
	f.PhoneNumber = &phone
	s := person("Vợ", domain.GenderFemale, domain.MemberKindInLaw)
	s.Generation = 1
	s.PhoneNumber = &phone // same number twice: the second insert violates the unique index

	_, err := r.genes.Create(ctx, owner, domain.CreateGenealogyParams{
		Name: "x", AncestralHome: "y", Founder: f, Spouses: []domain.GenealogyMember{s},
	}, 10)
	assert.ErrorIs(t, err, domain.ErrMemberPhoneInUse)

	list, err := r.genes.ListByUser(ctx, owner)
	require.NoError(t, err)
	assert.Empty(t, list, "a failed create leaves no tree, chronicle or member behind (AC12)")
}

func TestGenealogyRepository_RoleAndList(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	viewer := insertUser(t, r.db, "0900000002", "v@example.com")
	stranger := insertUser(t, r.db, "0900000003", "s@example.com")

	gid, _ := r.createTree(t, owner, "Họ A")
	_, err := r.db.Exec(ctx, `INSERT INTO genealogy_viewers (genealogy_id, user_id, granted_by) VALUES ($1, $2, $3)`, gid, viewer, owner)
	require.NoError(t, err)

	role, err := r.genes.ResolveRole(ctx, gid, owner)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleOwner, role)
	role, err = r.genes.ResolveRole(ctx, gid, viewer)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleViewer, role)

	_, err = r.genes.ResolveRole(ctx, gid, stranger)
	assert.ErrorIs(t, err, domain.ErrGenealogyNotFound, "no role looks the same as no such tree (AC15)")
	_, err = r.genes.ResolveRole(ctx, gid+999, owner)
	assert.ErrorIs(t, err, domain.ErrGenealogyNotFound)

	gid2, _ := r.createTree(t, viewer, "Họ B")
	list, err := r.genes.ListByUser(ctx, viewer)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, gid2, list[0].ID, "newest first")
	assert.Equal(t, domain.RoleOwner, list[0].MyRole)
	assert.Equal(t, domain.RoleViewer, list[1].MyRole)

	none, err := r.genes.ListByUser(ctx, stranger)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestGenealogyRepository_ChronicleUpsertAndDelete(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, _ := r.createTree(t, owner, "Họ A")

	c := &domain.Chronicle{
		GenealogyID: gid, Surname: strp("Nguyễn"), SinceYear: intp(1850),
		ChapYear: intp(2026), ChapMonth: intp(1), ChapDay: intp(31), ChapEnteredAsLunar: true,
		CoupletUpper: strp("上聯"), CoupletUpperReading: strp("thượng liên"), CoupletUpperMeaning: strp("vế trên"),
	}
	require.NoError(t, r.genes.UpsertChronicle(ctx, c))

	d, err := r.genes.FindDetail(ctx, gid)
	require.NoError(t, err)
	assert.Equal(t, "Nguyễn", *d.Chronicle.Surname)
	assert.Equal(t, 31, *d.Chronicle.ChapDay)
	assert.True(t, d.Chronicle.ChapEnteredAsLunar)
	assert.Equal(t, "thượng liên", *d.Chronicle.CoupletUpperReading)

	// Replacing: an omitted field is cleared.
	require.NoError(t, r.genes.UpsertChronicle(ctx, &domain.Chronicle{GenealogyID: gid, Origin: strp("Hà Nam")}))
	d, err = r.genes.FindDetail(ctx, gid)
	require.NoError(t, err)
	assert.Nil(t, d.Chronicle.Surname)
	assert.Nil(t, d.Chronicle.ChapYear)
	assert.False(t, d.Chronicle.ChapEnteredAsLunar)
	assert.Equal(t, "Hà Nam", *d.Chronicle.Origin)

	// The schema rejects a half-given chạp date.
	err = r.genes.UpsertChronicle(ctx, &domain.Chronicle{GenealogyID: gid, ChapYear: intp(2026)})
	assert.Error(t, err)

	require.NoError(t, r.genes.UpdateInfo(ctx, gid, "Họ Mới", "Hà Nội"))
	d, _ = r.genes.FindDetail(ctx, gid)
	assert.Equal(t, "Họ Mới", d.Name)
	assert.ErrorIs(t, r.genes.UpdateInfo(ctx, gid+999, "x", "y"), domain.ErrGenealogyNotFound)

	require.NoError(t, r.genes.Delete(ctx, gid))
	_, err = r.genes.FindDetail(ctx, gid)
	assert.ErrorIs(t, err, domain.ErrGenealogyNotFound)
	assert.ErrorIs(t, r.genes.Delete(ctx, gid), domain.ErrGenealogyNotFound)
	assert.ErrorIs(t, r.genes.UpsertChronicle(ctx, &domain.Chronicle{GenealogyID: gid}), domain.ErrGenealogyNotFound)
}

func TestGenealogyMemberRepository_AddChildAndSpouse(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	a := r.addChild(t, gid, root, "A", domain.GenderMale)
	b := r.addChild(t, gid, root, "B", domain.GenderMale)
	assert.Equal(t, 2, a.Generation)
	require.NotNil(t, a.BirthOrder)
	assert.Equal(t, 1, *a.BirthOrder, "first child defaults to birth order 1")
	assert.Equal(t, 2, *b.BirthOrder, "next child defaults to max + 1")

	w := r.addSpouse(t, gid, a.ID, "Vợ A", domain.GenderFemale)
	assert.Equal(t, domain.MemberKindInLaw, w.Kind)
	assert.Equal(t, 2, w.Generation, "an in-law shares the generation")
	require.NotNil(t, w.SpouseOfID)
	assert.Equal(t, a.ID, *w.SpouseOfID)
	assert.Nil(t, w.BirthOrder)

	// A grandchild naming the wife as the other parent.
	c := person("C", domain.GenderMale, domain.MemberKindBlood)
	c.OtherParentID = &w.ID
	gc, err := r.members.Add(ctx, gid, domain.RelationChild, a.ID, &c, genLimits())
	require.NoError(t, err)
	assert.Equal(t, 3, gc.Generation)
	require.NotNil(t, gc.OtherParentID)
	assert.Equal(t, w.ID, *gc.OtherParentID)
}

func TestGenealogyMemberRepository_AddRules(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")
	other := insertUser(t, r.db, "0900000002", "p@example.com")
	gid2, root2 := r.createTree(t, other, "Họ B")

	daughter := r.addChild(t, gid, root, "Mai", domain.GenderFemale)
	son := r.addChild(t, gid, root, "A", domain.GenderMale)
	wifeOfSon := r.addSpouse(t, gid, son.ID, "Vợ A", domain.GenderFemale)

	child := func() *domain.GenealogyMember { m := person("x", domain.GenderMale, domain.MemberKindBlood); return &m }

	t.Run("a related member of another tree is not found", func(t *testing.T) {
		_, err := r.members.Add(ctx, gid, domain.RelationChild, root2, child(), genLimits())
		assert.ErrorIs(t, err, domain.ErrMemberNotFound)
	})
	t.Run("no child under a daughter", func(t *testing.T) {
		_, err := r.members.Add(ctx, gid, domain.RelationChild, daughter.ID, child(), genLimits())
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
	})
	t.Run("no child under an in-law", func(t *testing.T) {
		_, err := r.members.Add(ctx, gid, domain.RelationChild, wifeOfSon.ID, child(), genLimits())
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
	})
	t.Run("other parent must be a spouse of the parent", func(t *testing.T) {
		m := child()
		m.OtherParentID = &wifeOfSon.ID // wife of A, but the parent is the founder
		_, err := r.members.Add(ctx, gid, domain.RelationChild, root, m, genLimits())
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "other_parent_id", ve.Field)
	})
	t.Run("member cap", func(t *testing.T) {
		lim := genLimits()
		lim.MaxMembers = 4 // founder, daughter, son, wife already
		_, err := r.members.Add(ctx, gid, domain.RelationChild, root, child(), lim)
		assert.ErrorIs(t, err, domain.ErrMemberLimitReached)
	})
	t.Run("generation cap", func(t *testing.T) {
		lim := genLimits()
		lim.MaxGeneration = 2
		_, err := r.members.Add(ctx, gid, domain.RelationChild, son.ID, child(), lim) // would be generation 3
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
	})
	t.Run("spouse cap", func(t *testing.T) {
		lim := genLimits()
		lim.MaxSpouses = 1
		sp := person("Vợ hai", domain.GenderFemale, domain.MemberKindInLaw)
		_, err := r.members.Add(ctx, gid, domain.RelationSpouse, son.ID, &sp, lim)
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
	})
	_ = gid2
}

func TestGenealogyMemberRepository_ConcurrentAddNeverExceedsTheMemberCap(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	lim := genLimits()
	lim.MaxMembers = 4 // founder + at most 3 more
	var wg sync.WaitGroup
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := person("x", domain.GenderMale, domain.MemberKindBlood)
			_, err := r.members.Add(ctx, gid, domain.RelationChild, root, &m, lim)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok := 0
	for err := range results {
		if err == nil {
			ok++
		}
	}
	assert.Equal(t, 3, ok)
	all, err := r.members.List(ctx, gid)
	require.NoError(t, err)
	assert.Len(t, all, 4)

	// Birth orders are 1..3 with no duplicates despite the concurrency.
	seen := map[int]bool{}
	for _, m := range all {
		if m.BirthOrder != nil {
			assert.False(t, seen[*m.BirthOrder], "birth_order %d duplicated", *m.BirthOrder)
			seen[*m.BirthOrder] = true
		}
	}
}

func TestGenealogyMemberRepository_PhoneIsUniquePerTreeAndFreePerOtherTree(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")
	gid2, root2 := r.createTree(t, owner, "Họ B")

	phone := "+84922222222"
	a := person("A", domain.GenderMale, domain.MemberKindBlood)
	a.PhoneNumber = &phone
	_, err := r.members.Add(ctx, gid, domain.RelationChild, root, &a, genLimits())
	require.NoError(t, err)

	b := person("B", domain.GenderMale, domain.MemberKindBlood)
	b.PhoneNumber = &phone
	_, err = r.members.Add(ctx, gid, domain.RelationChild, root, &b, genLimits())
	assert.ErrorIs(t, err, domain.ErrMemberPhoneInUse)

	c := person("C", domain.GenderMale, domain.MemberKindBlood)
	c.PhoneNumber = &phone
	_, err = r.members.Add(ctx, gid2, domain.RelationChild, root2, &c, genLimits())
	require.NoError(t, err, "the same number may appear in another tree")

	id, err := r.members.FindMemberIDByPhone(ctx, gid, phone)
	require.NoError(t, err)
	require.NotNil(t, id)
	none, err := r.members.FindMemberIDByPhone(ctx, gid, "+84900000000")
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestGenealogyMemberRepository_ClanHeadSwap(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	a := person("A", domain.GenderMale, domain.MemberKindBlood)
	a.IsClanHead = true
	ga, err := r.members.Add(ctx, gid, domain.RelationChild, root, &a, genLimits())
	require.NoError(t, err)
	assert.True(t, ga.IsClanHead)

	b := person("B", domain.GenderMale, domain.MemberKindBlood)
	b.IsClanHead = true
	gb, err := r.members.Add(ctx, gid, domain.RelationChild, root, &b, genLimits())
	require.NoError(t, err)
	assert.True(t, gb.IsClanHead)

	d, err := r.genes.FindDetail(ctx, gid)
	require.NoError(t, err)
	require.NotNil(t, d.ClanHeadMemberID)
	assert.Equal(t, gb.ID, *d.ClanHeadMemberID, "the new head replaces the old one")
	reload, err := r.members.FindByID(ctx, gid, ga.ID)
	require.NoError(t, err)
	assert.False(t, reload.IsClanHead)

	// Updating A to be head again swaps it back.
	reload.IsClanHead = true
	_, err = r.members.Update(ctx, gid, reload)
	require.NoError(t, err)
	d, _ = r.genes.FindDetail(ctx, gid)
	assert.Equal(t, ga.ID, *d.ClanHeadMemberID)
	reB, _ := r.members.FindByID(ctx, gid, gb.ID)
	assert.False(t, reB.IsClanHead)

	// Clearing the flag leaves no clan head.
	reload.IsClanHead = false
	_, err = r.members.Update(ctx, gid, reload)
	require.NoError(t, err)
	d, _ = r.genes.FindDetail(ctx, gid)
	assert.Nil(t, d.ClanHeadMemberID)
}

func TestGenealogyMemberRepository_Delete(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	a := r.addChild(t, gid, root, "A", domain.GenderMale)
	wife := r.addSpouse(t, gid, a.ID, "Vợ A", domain.GenderFemale)
	c := person("C", domain.GenderMale, domain.MemberKindBlood)
	c.OtherParentID = &wife.ID
	child, err := r.members.Add(ctx, gid, domain.RelationChild, a.ID, &c, genLimits())
	require.NoError(t, err)

	assert.ErrorIs(t, r.members.Delete(ctx, gid, root), domain.ErrFounderNotDeletable)
	assert.ErrorIs(t, r.members.Delete(ctx, gid, a.ID), domain.ErrMemberHasDescendants)
	assert.ErrorIs(t, r.members.Delete(ctx, gid, 999999), domain.ErrMemberNotFound)

	// Deleting the in-law only: the child keeps its blood parent and loses the link (AC58).
	require.NoError(t, r.members.Delete(ctx, gid, wife.ID))
	got, err := r.members.FindByID(ctx, gid, child.ID)
	require.NoError(t, err)
	assert.Nil(t, got.OtherParentID)
	require.NotNil(t, got.ParentID)

	// A leaf is deleted together with its in-laws.
	w2 := r.addSpouse(t, gid, child.ID, "Vợ C", domain.GenderFemale)
	require.NoError(t, r.members.Delete(ctx, gid, child.ID))
	_, err = r.members.FindByID(ctx, gid, w2.ID)
	assert.ErrorIs(t, err, domain.ErrMemberNotFound, "in-laws go with the member")
}

func TestGenealogyMemberRepository_Move(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	// root ── a ── a1 ── a11
	//      └─ b
	a := r.addChild(t, gid, root, "A", domain.GenderMale)
	b := r.addChild(t, gid, root, "B", domain.GenderMale)
	a1 := r.addChild(t, gid, a.ID, "A1", domain.GenderMale)
	a11 := r.addChild(t, gid, a1.ID, "A11", domain.GenderMale)
	wifeA1 := r.addSpouse(t, gid, a1.ID, "Vợ A1", domain.GenderFemale)
	b1 := r.addChild(t, gid, b.ID, "B1", domain.GenderMale)

	gen := func(id int64) int {
		m, err := r.members.FindByID(ctx, gid, id)
		require.NoError(t, err)
		return m.Generation
	}
	require.Equal(t, 3, gen(a1.ID))
	require.Equal(t, 4, gen(a11.ID))

	t.Run("moving a branch shifts the generation of its descendants and in-laws", func(t *testing.T) {
		// Move A1 (with A11 and the wife) under B1 (generation 3): A1 becomes 4.
		moved, err := r.members.Move(ctx, gid, a1.ID, domain.MoveMemberInput{Relation: "child", RelatedMemberID: b1.ID}, genLimits())
		require.NoError(t, err)
		assert.Equal(t, 4, moved.Generation)
		require.NotNil(t, moved.ParentID)
		assert.Equal(t, b1.ID, *moved.ParentID)
		assert.Equal(t, 5, gen(a11.ID))
		assert.Equal(t, 4, gen(wifeA1.ID), "the in-law follows the member it is attached to")
	})

	t.Run("a loop is refused", func(t *testing.T) {
		_, err := r.members.Move(ctx, gid, a1.ID, domain.MoveMemberInput{Relation: "child", RelatedMemberID: a11.ID}, genLimits())
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
		assert.Equal(t, 4, gen(a1.ID), "nothing changed")
	})

	t.Run("the founder cannot be moved", func(t *testing.T) {
		_, err := r.members.Move(ctx, gid, root, domain.MoveMemberInput{Relation: "child", RelatedMemberID: a.ID}, genLimits())
		assert.ErrorIs(t, err, domain.ErrFounderNotMovable)
	})

	t.Run("a move past the generation cap changes nothing", func(t *testing.T) {
		lim := genLimits()
		lim.MaxGeneration = 3 // A (generation 2) under B1 (generation 3) would land in generation 4
		_, err := r.members.Move(ctx, gid, a.ID, domain.MoveMemberInput{Relation: "child", RelatedMemberID: b1.ID}, lim)
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "related_member_id", ve.Field)
		assert.Equal(t, 2, gen(a.ID))
	})

	t.Run("relation must match the kind", func(t *testing.T) {
		_, err := r.members.Move(ctx, gid, wifeA1.ID, domain.MoveMemberInput{Relation: "child", RelatedMemberID: b.ID}, genLimits())
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "relation", ve.Field)
		_, err = r.members.Move(ctx, gid, a11.ID, domain.MoveMemberInput{Relation: "spouse", RelatedMemberID: b.ID}, genLimits())
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "relation", ve.Field)
	})

	t.Run("an in-law moves to another blood member and the children lose the link", func(t *testing.T) {
		// A11's mother is recorded as the wife of A1.
		m := person("A12", domain.GenderMale, domain.MemberKindBlood)
		m.OtherParentID = &wifeA1.ID
		kid, err := r.members.Add(ctx, gid, domain.RelationChild, a1.ID, &m, genLimits())
		require.NoError(t, err)

		moved, err := r.members.Move(ctx, gid, wifeA1.ID, domain.MoveMemberInput{Relation: "spouse", RelatedMemberID: b.ID}, genLimits())
		require.NoError(t, err)
		require.NotNil(t, moved.SpouseOfID)
		assert.Equal(t, b.ID, *moved.SpouseOfID)
		assert.Equal(t, gen(b.ID), moved.Generation)
		got, err := r.members.FindByID(ctx, gid, kid.ID)
		require.NoError(t, err)
		assert.Nil(t, got.OtherParentID)
	})

	t.Run("an unknown member or target is not found", func(t *testing.T) {
		_, err := r.members.Move(ctx, gid, 999999, domain.MoveMemberInput{Relation: "child", RelatedMemberID: b.ID}, genLimits())
		assert.ErrorIs(t, err, domain.ErrMemberNotFound)
		_, err = r.members.Move(ctx, gid, a.ID, domain.MoveMemberInput{Relation: "child", RelatedMemberID: 999999}, genLimits())
		assert.ErrorIs(t, err, domain.ErrMemberNotFound)
	})
}

func TestGenealogyInvitations_FullJourney(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	invitee := insertUser(t, r.db, "0900000002", "i@example.com")
	gid, root := r.createTree(t, owner, "Họ A")
	son := r.addChild(t, gid, root, "A", domain.GenderMale)

	// phone numbers in users are stored as typed in these fixtures
	phone := "0900000002"
	ttl := 30 * 24 * time.Hour

	inv, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: son.ID, PhoneNumber: phone}, 20, ttl)
	require.NoError(t, err)
	assert.Equal(t, domain.InvitationPending, inv.Status)
	assert.Equal(t, son.ID, inv.MemberID)
	assert.Equal(t, "A", inv.MemberFullName)
	assert.True(t, inv.ExpiresAt.After(time.Now().Add(29*24*time.Hour)))

	// The number was written onto the member.
	onNode, err := r.members.FindByID(ctx, gid, son.ID)
	require.NoError(t, err)
	require.NotNil(t, onNode.PhoneNumber)
	assert.Equal(t, phone, *onNode.PhoneNumber)

	t.Run("the invitee sees it, a stranger does not", func(t *testing.T) {
		mine, err := r.invitations.ListMine(ctx, phone)
		require.NoError(t, err)
		require.Len(t, mine, 1)
		assert.Equal(t, "Họ A", mine[0].GenealogyName)
		assert.Equal(t, "A", mine[0].MemberFullName)
		assert.Equal(t, "0900000001", mine[0].InvitedByPhone)
		other, err := r.invitations.ListMine(ctx, "0911111111")
		require.NoError(t, err)
		assert.Empty(t, other)
	})

	t.Run("a second invitation for the same number is refused", func(t *testing.T) {
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: son.ID}, 20, ttl)
		assert.ErrorIs(t, err, domain.ErrInvitationAlreadyPending)
	})

	t.Run("someone else cannot accept or decline it", func(t *testing.T) {
		_, err := r.invitations.Accept(ctx, inv.ID, invitee, "0911111111", 20)
		assert.ErrorIs(t, err, domain.ErrInvitationNotFound)
		assert.ErrorIs(t, r.invitations.Decline(ctx, inv.ID, "0911111111"), domain.ErrInvitationNotFound)
	})

	t.Run("accepting grants viewer access", func(t *testing.T) {
		res, err := r.invitations.Accept(ctx, inv.ID, invitee, phone, 20)
		require.NoError(t, err)
		assert.Equal(t, gid, res.GenealogyID)
		assert.Equal(t, son.ID, res.MemberID)

		role, err := r.genes.ResolveRole(ctx, gid, invitee)
		require.NoError(t, err)
		assert.Equal(t, domain.RoleViewer, role)

		viewers, err := r.viewers.List(ctx, gid)
		require.NoError(t, err)
		require.Len(t, viewers, 1)
		require.NotNil(t, viewers[0].MemberID)
		assert.Equal(t, son.ID, *viewers[0].MemberID, "the viewer stands at the invited member (AC48)")

		list, err := r.invitations.ListByGenealogy(ctx, gid)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, domain.InvitationAccepted, list[0].Status, "the owner sees the status change")

		// A second accept or a decline now finds it no longer pending.
		_, err = r.invitations.Accept(ctx, inv.ID, invitee, phone, 20)
		assert.ErrorIs(t, err, domain.ErrInvitationNotPending)
		assert.ErrorIs(t, r.invitations.Decline(ctx, inv.ID, phone), domain.ErrInvitationNotPending)
	})

	t.Run("an existing viewer or the owner is refused a new invitation", func(t *testing.T) {
		b := r.addChild(t, gid, root, "B", domain.GenderMale)
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: b.ID, PhoneNumber: phone}, 20, ttl)
		assert.ErrorIs(t, err, domain.ErrViewerExists, "the number belongs to a user who already views this tree")
		_, err = r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: b.ID, PhoneNumber: "0900000001"}, 20, ttl)
		assert.ErrorIs(t, err, domain.ErrViewerExists, "the owner's own number")

		// A number that sits on another member, with no one holding it yet, is a
		// uniqueness conflict rather than an access conflict.
		held := "0988888888"
		x := person("X", domain.GenderMale, domain.MemberKindBlood)
		x.PhoneNumber = &held
		_, err = r.members.Add(ctx, gid, domain.RelationChild, root, &x, genLimits())
		require.NoError(t, err)
		_, err = r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: b.ID, PhoneNumber: held}, 20, ttl)
		assert.ErrorIs(t, err, domain.ErrMemberPhoneInUse)
	})

	t.Run("a viewer can be removed and is then a stranger", func(t *testing.T) {
		require.NoError(t, r.viewers.Remove(ctx, gid, invitee))
		_, err := r.genes.ResolveRole(ctx, gid, invitee)
		assert.ErrorIs(t, err, domain.ErrGenealogyNotFound)
		assert.ErrorIs(t, r.viewers.Remove(ctx, gid, invitee), domain.ErrViewerNotFound)
	})
}

func TestGenealogyInvitations_Rules(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")
	ttl := 30 * 24 * time.Hour

	a := r.addChild(t, gid, root, "A", domain.GenderMale)
	dead := person("Cụ", domain.GenderMale, domain.MemberKindBlood)
	dead.IsAlive = false
	deceased, err := r.members.Add(ctx, gid, domain.RelationChild, root, &dead, genLimits())
	require.NoError(t, err)

	t.Run("a deceased member cannot be invited", func(t *testing.T) {
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: deceased.ID, PhoneNumber: "0911111111"}, 20, ttl)
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "member_id", ve.Field)
	})

	t.Run("a number is required when the member has none", func(t *testing.T) {
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: a.ID}, 20, ttl)
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "phone_number", ve.Field)
	})

	t.Run("an unknown member is not found", func(t *testing.T) {
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: 999999, PhoneNumber: "0911111111"}, 20, ttl)
		assert.ErrorIs(t, err, domain.ErrMemberNotFound)
	})

	t.Run("a number that differs from the one on the member is refused", func(t *testing.T) {
		p := "0933333333"
		m := person("D", domain.GenderMale, domain.MemberKindBlood)
		m.PhoneNumber = &p
		d, err := r.members.Add(ctx, gid, domain.RelationChild, root, &m, genLimits())
		require.NoError(t, err)
		_, err = r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: d.ID, PhoneNumber: "0944444444"}, 20, ttl)
		var ve *domain.GenealogyValidationError
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "phone_number", ve.Field)

		// Without typing a number, the one on the member is used.
		inv, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: d.ID}, 20, ttl)
		require.NoError(t, err)
		assert.Equal(t, p, inv.PhoneNumber)
	})

	t.Run("the cap counts pending invitations", func(t *testing.T) {
		_, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: a.ID, PhoneNumber: "0955555555"}, 1, ttl)
		// one pending invitation already exists (for D), so a cap of 1 is full
		assert.ErrorIs(t, err, domain.ErrViewerLimitReached)
	})

	t.Run("an expired invitation can be replaced and is not acceptable", func(t *testing.T) {
		b := r.addChild(t, gid, root, "B", domain.GenderMale)
		first, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: b.ID, PhoneNumber: "0966666666"}, 20, ttl)
		require.NoError(t, err)
		_, err = r.db.Exec(ctx, `UPDATE genealogy_invitations SET expires_at = now() - interval '1 hour' WHERE id = $1`, first.ID)
		require.NoError(t, err)

		mine, err := r.invitations.ListMine(ctx, "0966666666")
		require.NoError(t, err)
		assert.Empty(t, mine, "an expired invitation is not in the inbox")

		invitee := insertUser(t, r.db, "0966666666", "late@example.com")
		_, err = r.invitations.Accept(ctx, first.ID, invitee, "0966666666", 20)
		assert.ErrorIs(t, err, domain.ErrInvitationNotPending)

		second, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: b.ID}, 20, ttl)
		require.NoError(t, err, "the lapsed invitation does not block a new one")
		assert.NotEqual(t, first.ID, second.ID)

		list, err := r.invitations.ListByGenealogy(ctx, gid)
		require.NoError(t, err)
		statuses := map[int64]domain.InvitationStatus{}
		for _, i := range list {
			statuses[i.ID] = i.Status
		}
		assert.Equal(t, domain.InvitationExpired, statuses[first.ID])
		assert.Equal(t, domain.InvitationPending, statuses[second.ID])
	})

	t.Run("cancel hides the invitation; clearing a number cancels it", func(t *testing.T) {
		c := r.addChild(t, gid, root, "C", domain.GenderMale)
		inv, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: c.ID, PhoneNumber: "0977777777"}, 20, ttl)
		require.NoError(t, err)

		// The owner clears the member's number: the pending invitation is cancelled (AC30).
		node, err := r.members.FindByID(ctx, gid, c.ID)
		require.NoError(t, err)
		node.PhoneNumber = nil
		_, err = r.members.Update(ctx, gid, node)
		require.NoError(t, err)

		mine, err := r.invitations.ListMine(ctx, "0977777777")
		require.NoError(t, err)
		assert.Empty(t, mine)
		assert.ErrorIs(t, r.invitations.Cancel(ctx, gid, inv.ID), domain.ErrInvitationNotFound, "already cancelled")

		list, err := r.invitations.ListByGenealogy(ctx, gid)
		require.NoError(t, err)
		for _, i := range list {
			assert.NotEqual(t, inv.ID, i.ID, "a cancelled invitation is hidden from the owner's list")
		}
	})
}

func TestGenealogyInvitations_AcceptConcurrentlyNeverExceedsTheCap(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")
	ttl := 30 * 24 * time.Hour

	const invitees, cap = 6, 2
	type pair struct {
		invID, userID int64
		phone         string
	}
	var pairs []pair
	for i := 0; i < invitees; i++ {
		phone := "09100000" + string(rune('0'+i)) + "0"
		uid := insertUser(t, r.db, phone, phone+"@example.com")
		m := r.addChild(t, gid, root, "M", domain.GenderMale)
		// Big cap for the invitations, small cap enforced at accept time below.
		inv, err := r.invitations.Create(ctx, gid, owner, domain.InviteInput{MemberID: m.ID, PhoneNumber: phone}, 100, ttl)
		require.NoError(t, err)
		pairs = append(pairs, pair{inv.ID, uid, phone})
	}

	var wg sync.WaitGroup
	results := make(chan error, invitees)
	for _, p := range pairs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.invitations.Accept(ctx, p.invID, p.userID, p.phone, cap)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok := 0
	for err := range results {
		if err == nil {
			ok++
		}
	}
	assert.Equal(t, cap, ok, "exactly cap accepts succeed, and none deadlock")
	viewers, err := r.viewers.List(ctx, gid)
	require.NoError(t, err)
	assert.Len(t, viewers, cap)
}

func TestUserPhoneLookup(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	uid := insertUser(t, r.db, "0900000001", "o@example.com")

	phone, err := r.phones.FindPhoneByID(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "0900000001", phone)

	_, err = r.phones.FindPhoneByID(ctx, uid+999)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

// AC32: changing a blood member's gender leaves its in-laws attached; only the
// derived DAU/RE label would change, and the label is not stored.
func TestGenealogyMemberRepository_GenderChangeKeepsInLawsAttached(t *testing.T) {
	r := newGenRepos(t)
	ctx := context.Background()
	owner := insertUser(t, r.db, "0900000001", "o@example.com")
	gid, root := r.createTree(t, owner, "Họ A")

	a := r.addChild(t, gid, root, "A", domain.GenderMale)
	wife := r.addSpouse(t, gid, a.ID, "Vợ A", domain.GenderFemale)

	a.Gender = domain.GenderFemale
	_, err := r.members.Update(ctx, gid, a)
	require.NoError(t, err)

	got, err := r.members.FindByID(ctx, gid, wife.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SpouseOfID)
	assert.Equal(t, a.ID, *got.SpouseOfID, "the in-law is still attached after the gender change")
	assert.Equal(t, domain.MemberKindInLaw, got.Kind)
}
